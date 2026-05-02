package service

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	ginKeyRetryExcludedChannels = "retry_excluded_channels"
	ginKeyRoutingCooldownInfo   = "routing_cooldown_info"

	channelRoutingCooldownKindQuota     = "quota"
	channelRoutingCooldownKindRateLimit = "rate_limit"

	channelRoutingCooldownSourceHeader     = "header"
	channelRoutingCooldownSourceMetadata   = "metadata"
	channelRoutingCooldownSourceUsageProbe = "usage_probe"
	channelRoutingCooldownSourceFallback   = "fallback"
	channelRoutingCooldownSourceDailyQuota = "daily_quota_fallback"

	defaultChannelRoutingCooldown = 15 * time.Minute
	dailyQuotaResetDelay          = 5 * time.Minute
)

type routingCooldownAdminInfo struct {
	Applied  bool   `json:"applied"`
	Kind     string `json:"kind,omitempty"`
	Source   string `json:"source,omitempty"`
	ResetAt  int64  `json:"reset_at,omitempty"`
	KeyIndex *int   `json:"key_index,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func AddRetryExcludedChannel(c *gin.Context, channelID int) {
	if c == nil || channelID <= 0 {
		return
	}
	existing := GetRetryExcludedChannels(c)
	for _, id := range existing {
		if id == channelID {
			return
		}
	}
	existing = append(existing, channelID)
	sort.Ints(existing)
	c.Set(ginKeyRetryExcludedChannels, existing)
}

func GetRetryExcludedChannels(c *gin.Context) []int {
	if c == nil {
		return nil
	}
	value, ok := c.Get(ginKeyRetryExcludedChannels)
	if !ok {
		return nil
	}
	if ids, ok := value.([]int); ok {
		return ids
	}
	return nil
}

func ShouldExcludeChannelAfterFailure(channelError types.ChannelError, _ *gin.Context) bool {
	return channelError.ChannelId > 0
}

func AppendChannelRoutingAdminInfo(c *gin.Context, adminInfo map[string]interface{}) {
	if c == nil || adminInfo == nil {
		return
	}
	if excluded := GetRetryExcludedChannels(c); len(excluded) > 0 {
		adminInfo["retry_excluded_channels"] = excluded
	}
	info, ok := getRoutingCooldownAdminInfo(c)
	if ok && info.Applied {
		payload := map[string]interface{}{
			"applied":  true,
			"kind":     info.Kind,
			"source":   info.Source,
			"reset_at": info.ResetAt,
			"reason":   info.Reason,
		}
		if info.KeyIndex != nil {
			payload["key_index"] = *info.KeyIndex
		}
		adminInfo["routing_cooldown"] = payload
	}
}

func HandleChannelRoutingCooldown(c *gin.Context, channelError types.ChannelError, err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	if HandleCodexQuotaChannelError(c, channelError, err) {
		return true
	}

	state, ok := classifyGenericRoutingCooldown(channelError.ChannelType, err)
	if !ok {
		return false
	}
	if persistErr := persistChannelRoutingCooldown(channelError, state); persistErr != nil {
		common.SysLog(fmt.Sprintf("routing cooldown persist failed: channel_id=%d, error=%v", channelError.ChannelId, persistErr))
		return false
	}
	setRoutingCooldownAdminInfo(c, state)
	return true
}

func classifyGenericRoutingCooldown(channelType int, err *types.NewAPIError) (model.RoutingCooldownState, bool) {
	if err == nil || types.IsSkipRetryError(err) {
		return model.RoutingCooldownState{}, false
	}

	lowerCode := strings.ToLower(string(err.GetErrorCode()))
	lowerMessage := strings.ToLower(err.Error())
	hasQuotaSignal := isQuotaExhaustionMessage(lowerCode, lowerMessage)
	isRateLimitStatus := err.StatusCode == http.StatusTooManyRequests
	isForbiddenQuotaStatus := err.StatusCode == http.StatusForbidden &&
		hasQuotaSignal &&
		isOpenAIQuotaCooldownChannel(channelType)
	if !isRateLimitStatus && !isForbiddenQuotaStatus {
		return model.RoutingCooldownState{}, false
	}

	resetAt, source, hasReset := resolveChannelRoutingCooldownReset(err.Metadata)
	if !hasReset && !isRateLimitMessage(lowerCode, lowerMessage) && !hasQuotaSignal {
		return model.RoutingCooldownState{}, false
	}
	if !hasReset {
		if isDailyQuotaMessage(lowerCode, lowerMessage) || (hasQuotaSignal && err.StatusCode == http.StatusForbidden) {
			resetAt = nextDailyQuotaReset(time.Now()).Unix()
			source = channelRoutingCooldownSourceDailyQuota
		} else {
			resetAt = common.GetTimestamp() + int64(defaultChannelRoutingCooldown/time.Second)
			source = channelRoutingCooldownSourceFallback
		}
	}

	kind := channelRoutingCooldownKindRateLimit
	if hasQuotaSignal {
		kind = channelRoutingCooldownKindQuota
	}

	return model.RoutingCooldownState{
		Kind:      kind,
		Reason:    err.ErrorWithStatusCode(),
		ResetAt:   resetAt,
		Source:    source,
		CreatedAt: common.GetTimestamp(),
	}, true
}

func resolveChannelRoutingCooldownReset(metadata []byte) (int64, string, bool) {
	if len(metadata) == 0 || common.GetJsonType(metadata) != "object" {
		return 0, "", false
	}

	meta := make(map[string]interface{})
	if err := common.Unmarshal(metadata, &meta); err != nil {
		return 0, "", false
	}
	now := time.Now()
	if resetAt := extractCodexQuotaResetAt(meta, now); resetAt > 0 {
		source := channelRoutingCooldownSourceMetadata
		if _, ok := meta["retry_after"]; ok {
			source = channelRoutingCooldownSourceHeader
		} else if _, ok := meta["rate_limit_reset"]; ok {
			source = channelRoutingCooldownSourceHeader
		}
		return resetAt, source, true
	}
	return 0, "", false
}

func persistChannelRoutingCooldown(channelError types.ChannelError, state model.RoutingCooldownState) error {
	channel, err := model.GetChannelById(channelError.ChannelId, true)
	if err != nil {
		return err
	}

	keyIndex := -1
	if channel.ChannelInfo.IsMultiKey && channelError.UsingKeyIdx >= 0 {
		keyIndex = channelError.UsingKeyIdx
	}
	state.KeyIndex = keyIndex

	states := channel.GetRoutingCooldownStates()
	states[strconv.Itoa(keyIndex)] = state
	channel.SetRoutingCooldownStates(states)
	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	if common.MemoryCacheEnabled {
		model.CacheUpdateChannel(channel)
	}
	return nil
}

func getRoutingCooldownAdminInfo(c *gin.Context) (routingCooldownAdminInfo, bool) {
	if c == nil {
		return routingCooldownAdminInfo{}, false
	}
	value, ok := c.Get(ginKeyRoutingCooldownInfo)
	if !ok {
		return routingCooldownAdminInfo{}, false
	}
	info, ok := value.(routingCooldownAdminInfo)
	if !ok {
		return routingCooldownAdminInfo{}, false
	}
	return info, true
}

func setRoutingCooldownAdminInfo(c *gin.Context, state model.RoutingCooldownState) {
	if c == nil {
		return
	}
	info := routingCooldownAdminInfo{
		Applied: true,
		Kind:    state.Kind,
		Source:  state.Source,
		ResetAt: state.ResetAt,
		Reason:  state.Reason,
	}
	if state.KeyIndex >= 0 {
		keyIndex := state.KeyIndex
		info.KeyIndex = &keyIndex
	}
	c.Set(ginKeyRoutingCooldownInfo, info)
}

func isRateLimitMessage(lowerCode string, lowerMessage string) bool {
	if strings.Contains(lowerCode, "rate_limit") || strings.Contains(lowerCode, "too_many_requests") {
		return true
	}
	for _, marker := range []string{
		"rate limit",
		"too many requests",
		"usage limit",
		"try again later",
		"quota",
	} {
		if strings.Contains(lowerMessage, marker) {
			return true
		}
	}
	return false
}

func isOpenAIQuotaCooldownChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI:
		return true
	default:
		return false
	}
}

func isQuotaExhaustionMessage(lowerCode string, lowerMessage string) bool {
	if strings.Contains(lowerCode, "quota") ||
		strings.Contains(lowerCode, "insufficient_quota") ||
		strings.Contains(lowerCode, "billing") {
		return true
	}
	for _, marker := range []string{
		"quota",
		"usage limit",
		"credit balance",
		"insufficient balance",
		"insufficient quota",
		"limit reached",
		"daily limit",
		"per day",
		"额度不足",
		"剩余额度",
		"余额不足",
		"可用额度不足",
		"日限额",
		"每日限额",
		"每天限制",
	} {
		if strings.Contains(lowerMessage, marker) {
			return true
		}
	}
	return false
}

func isDailyQuotaMessage(lowerCode string, lowerMessage string) bool {
	for _, marker := range []string{
		"daily_limit",
		"daily limit",
		"daily usage limit",
		"per day",
		"day quota",
		"日限额",
		"每日限额",
		"每天限制",
	} {
		if strings.Contains(lowerCode, marker) || strings.Contains(lowerMessage, marker) {
			return true
		}
	}
	return false
}

func nextDailyQuotaReset(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return nextDay.Add(dailyQuotaResetDelay)
}

func CleanupChannelRoutingCooldown(channelID int, now int64) error {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return err
	}
	if !channel.DropExpiredRoutingCooldowns(now) {
		return nil
	}
	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	if common.MemoryCacheEnabled {
		model.CacheUpdateChannel(channel)
	}
	return nil
}

func ClearChannelRoutingCooldown(channelID int) (*model.Channel, error) {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	channel.SetRoutingCooldownStates(map[string]model.RoutingCooldownState{})
	if err := channel.SaveWithoutKey(); err != nil {
		return nil, err
	}
	if common.MemoryCacheEnabled {
		model.CacheUpdateChannel(channel)
	}
	channel.ApplyRoutingCooldownView(common.GetTimestamp())
	return channel, nil
}

func RunChannelRoutingCooldownCleanupPass(now int64) error {
	var channels []*model.Channel
	if err := model.DB.Find(&channels).Error; err != nil {
		return err
	}
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		if err := CleanupChannelRoutingCooldown(channel.Id, now); err != nil {
			common.SysLog(fmt.Sprintf("routing cooldown cleanup failed: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return nil
}

func EnsureRetryTimesDefault() {
	if common.RetryTimes <= 0 {
		common.RetryTimes = 1
	}
}

func ResolveRetryBudget(retryTimes int) int {
	if retryTimes > 0 {
		return retryTimes
	}
	return 1
}

func ShouldForceRetryAfterAffinityRelease(c *gin.Context) bool {
	return ConsumeChannelAffinityForcedRetry(c)
}

func ShouldSkipRetryByChannelAffinity(c *gin.Context) bool {
	return ShouldSkipRetryAfterChannelAffinityFailure(c)
}

func ShouldRetryableStatus(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	if types.IsSkipRetryError(err) {
		return false
	}
	if operation_setting.IsAlwaysSkipRetryCode(err.GetErrorCode()) {
		return false
	}
	if types.IsChannelError(err) {
		return true
	}
	if err.StatusCode < 100 || err.StatusCode > 599 {
		return true
	}
	return operation_setting.ShouldRetryByStatusCode(err.StatusCode)
}
