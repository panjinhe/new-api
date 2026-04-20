package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

const (
	codexQuotaStateOtherInfoKey = "codex_quota_state"
	codexQuotaDisableKindQuota  = "quota"
	codexQuotaScopeFiveHour     = "five_hour"
	codexQuotaScopeWeekly       = "weekly"
	codexQuotaScopeUnknown      = "unknown"
	codexQuotaSourceError       = "error"
	codexQuotaSourceUsage       = "usage"
	codexQuotaSingleKeyIndex    = -1
)

var codexQuotaMessageKeywords = []string{
	"usage limit",
	"rate limit",
	"quota",
	"limit reached",
	"try again later",
	"too many requests",
	"weekly",
	"5 hour",
	"5-hour",
	"five hour",
}

var fetchCodexWhamUsageForChannelFunc = fetchCodexWhamUsageForChannel

type CodexQuotaState struct {
	DisableKind  string `json:"disable_kind"`
	Scope        string `json:"scope"`
	ResetAt      int64  `json:"reset_at,omitempty"`
	Source       string `json:"source,omitempty"`
	DisabledAt   int64  `json:"disabled_at"`
	ResetUnknown bool   `json:"reset_unknown,omitempty"`
}

type codexQuotaWindowCandidate struct {
	Scope       string
	ResetAt     int64
	UsedPercent float64
}

type codexWhamUsageResponse struct {
	RateLimit struct {
		PrimaryWindow   map[string]interface{} `json:"primary_window"`
		SecondaryWindow map[string]interface{} `json:"secondary_window"`
	} `json:"rate_limit"`
}

func HandleCodexQuotaChannelError(ctx context.Context, channelError types.ChannelError, err *types.NewAPIError) bool {
	if err == nil || !channelError.AutoBan || channelError.ChannelType != constant.ChannelTypeCodex {
		return false
	}

	state, ok := classifyCodexQuotaState(err)
	if !ok {
		return false
	}
	types.ErrOptionWithErrorCode(types.ErrorCodeChannelCodexQuotaExhausted)(err)

	if state.ResetAt <= 0 && !state.ResetUnknown {
		resolvedState, resolveErr := resolveCodexQuotaStateFromUsage(ctx, channelError.ChannelId, state.Scope)
		if resolveErr == nil && resolvedState != nil && resolvedState.ResetAt > 0 {
			state.ResetAt = resolvedState.ResetAt
			state.Scope = resolvedState.Scope
			state.Source = codexQuotaSourceUsage
		} else {
			state.ResetUnknown = true
		}
	}

	if state.Source == "" {
		state.Source = codexQuotaSourceError
	}
	if state.DisabledAt == 0 {
		state.DisabledAt = common.GetTimestamp()
	}

	if err := disableCodexChannelForQuota(channelError, err.ErrorWithStatusCode(), state); err != nil {
		common.SysLog(fmt.Sprintf("codex quota disable failed: channel_id=%d, error=%v", channelError.ChannelId, err))
		return false
	}
	return true
}

func ShouldSkipAutoTestForCodexQuota(channel *model.Channel, now int64) bool {
	if channel == nil || channel.Type != constant.ChannelTypeCodex || channel.Status != common.ChannelStatusAutoDisabled {
		return false
	}

	states := getCodexQuotaStates(channel)
	for _, state := range states {
		if state.DisableKind != codexQuotaDisableKindQuota {
			continue
		}
		if state.ResetUnknown || state.ResetAt == 0 || state.ResetAt > now {
			return true
		}
	}
	return false
}

func runCodexQuotaAutoReenablePass(ctx context.Context, now int64) error {
	statuses := []int{common.ChannelStatusEnabled, common.ChannelStatusAutoDisabled}
	var channels []*model.Channel
	if err := model.DB.Where("type = ? AND status IN ?", constant.ChannelTypeCodex, statuses).Find(&channels).Error; err != nil {
		return err
	}

	for _, channel := range channels {
		if channel == nil {
			continue
		}
		if err := recoverDueCodexQuotaStates(ctx, channel.Id, now); err != nil {
			common.SysLog(fmt.Sprintf("codex quota auto-reenable failed: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return nil
}

func recoverDueCodexQuotaStates(_ context.Context, channelID int, now int64) error {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return err
	}

	states := getCodexQuotaStates(channel)
	if len(states) == 0 {
		return nil
	}

	entryKeys := make([]string, 0, len(states))
	for entryKey, state := range states {
		if state.DisableKind != codexQuotaDisableKindQuota || state.ResetUnknown || state.ResetAt == 0 || state.ResetAt > now {
			continue
		}
		entryKeys = append(entryKeys, entryKey)
	}
	sort.Strings(entryKeys)
	if len(entryKeys) == 0 {
		return nil
	}

	for _, entryKey := range entryKeys {
		keyIndex, err := strconv.Atoi(entryKey)
		if err != nil {
			delete(states, entryKey)
			continue
		}

		if usingKey, ok := resolveCodexQuotaUsingKey(channel, keyIndex); ok {
			_ = model.UpdateChannelStatus(channel.Id, usingKey, common.ChannelStatusEnabled, "")
		}
		delete(states, entryKey)

		channel, err = model.GetChannelById(channelID, true)
		if err != nil {
			return err
		}
	}

	return saveCodexQuotaStates(channel, states)
}

func classifyCodexQuotaState(err *types.NewAPIError) (*CodexQuotaState, bool) {
	if err == nil {
		return nil, false
	}

	metadataState, metadataSignals := classifyCodexQuotaStateFromMetadata(err.Metadata, strings.ToLower(err.Error()))
	if metadataState != nil {
		return metadataState, true
	}

	if !metadataSignals && !isCodexQuotaMessage(strings.ToLower(err.Error())) {
		return nil, false
	}
	if err.StatusCode != http.StatusTooManyRequests && err.StatusCode != http.StatusForbidden {
		return nil, false
	}

	return &CodexQuotaState{
		DisableKind: codexQuotaDisableKindQuota,
		Scope:       classifyCodexQuotaScopeFromText(strings.ToLower(err.Error())),
		Source:      codexQuotaSourceError,
		DisabledAt:  common.GetTimestamp(),
	}, true
}

func classifyCodexQuotaStateFromMetadata(metadata []byte, lowerMessage string) (*CodexQuotaState, bool) {
	if len(metadata) == 0 || common.GetJsonType(metadata) != "object" {
		return nil, false
	}

	meta := make(map[string]interface{})
	if err := common.Unmarshal(metadata, &meta); err != nil {
		return nil, false
	}

	scope := classifyCodexQuotaScopeFromText(lowerMessage)
	windows := collectCodexQuotaWindows(meta, time.Now())
	if window := selectCodexQuotaWindow(windows, scope); window != nil {
		return &CodexQuotaState{
			DisableKind: codexQuotaDisableKindQuota,
			Scope:       window.Scope,
			ResetAt:     window.ResetAt,
			Source:      codexQuotaSourceError,
			DisabledAt:  common.GetTimestamp(),
		}, true
	}

	if scope == codexQuotaScopeUnknown {
		if limitWindowSeconds, ok := parseInt64(meta["limit_window_seconds"]); ok {
			scope = classifyCodexQuotaScopeByWindowDuration(limitWindowSeconds)
		}
	}

	resetAt := extractCodexQuotaResetAt(meta, time.Now())
	if resetAt == 0 {
		if !isCodexQuotaMessage(lowerMessage) {
			return nil, len(windows) > 0 || strings.Contains(strings.ToLower(string(metadata)), "limit")
		}
		return &CodexQuotaState{
			DisableKind: codexQuotaDisableKindQuota,
			Scope:       firstNonEmpty(scope, codexQuotaScopeUnknown),
			Source:      codexQuotaSourceError,
			DisabledAt:  common.GetTimestamp(),
		}, true
	}

	return &CodexQuotaState{
		DisableKind: codexQuotaDisableKindQuota,
		Scope:       firstNonEmpty(scope, codexQuotaScopeUnknown),
		ResetAt:     resetAt,
		Source:      codexQuotaSourceError,
		DisabledAt:  common.GetTimestamp(),
	}, true
}

func resolveCodexQuotaStateFromUsage(ctx context.Context, channelID int, preferredScope string) (*CodexQuotaState, error) {
	resolveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	statusCode, body, err := fetchCodexWhamUsageForChannelFunc(resolveCtx, channelID)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("codex usage upstream status=%d", statusCode)
	}

	var payload codexWhamUsageResponse
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	windows := make([]codexQuotaWindowCandidate, 0, 2)
	for _, raw := range []map[string]interface{}{payload.RateLimit.PrimaryWindow, payload.RateLimit.SecondaryWindow} {
		if raw == nil {
			continue
		}
		if window := parseCodexQuotaWindow(raw, time.Now()); window != nil {
			windows = append(windows, *window)
		}
	}

	window := selectCodexQuotaWindow(windows, preferredScope)
	if window == nil {
		return nil, errors.New("no exhausted codex quota window found")
	}

	return &CodexQuotaState{
		DisableKind: codexQuotaDisableKindQuota,
		Scope:       window.Scope,
		ResetAt:     window.ResetAt,
		Source:      codexQuotaSourceUsage,
		DisabledAt:  common.GetTimestamp(),
	}, nil
}

func fetchCodexWhamUsageForChannel(ctx context.Context, channelID int) (int, []byte, error) {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return 0, nil, err
	}

	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(channel.Key))
	if err != nil {
		return 0, nil, err
	}

	client, err := NewProxyHttpClient(channel.GetSetting().Proxy)
	if err != nil {
		return 0, nil, err
	}

	statusCode, body, err := FetchCodexWhamUsage(ctx, client, channel.GetBaseURL(), strings.TrimSpace(oauthKey.AccessToken), strings.TrimSpace(oauthKey.AccountID))
	if err != nil {
		return 0, nil, err
	}
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return statusCode, body, nil
	}
	if strings.TrimSpace(oauthKey.RefreshToken) == "" {
		return statusCode, body, nil
	}

	refreshedKey, refreshedChannel, refreshErr := RefreshCodexChannelCredential(ctx, channelID, CodexCredentialRefreshOptions{ResetCaches: true})
	if refreshErr != nil {
		return statusCode, body, nil
	}
	return FetchCodexWhamUsage(ctx, client, refreshedChannel.GetBaseURL(), strings.TrimSpace(refreshedKey.AccessToken), strings.TrimSpace(refreshedKey.AccountID))
}

func disableCodexChannelForQuota(channelError types.ChannelError, reason string, state *CodexQuotaState) error {
	if state == nil {
		return errors.New("nil codex quota state")
	}

	_ = model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, reason)
	if err := persistCodexQuotaState(channelError.ChannelId, resolveCodexQuotaStateIndex(channelError), *state); err != nil {
		return err
	}

	channel, err := model.GetChannelById(channelError.ChannelId, true)
	if err != nil {
		return nil
	}

	subject := fmt.Sprintf("通道「%s」（#%d）触发 Codex 限额", channelError.ChannelName, channelError.ChannelId)
	content := fmt.Sprintf("通道「%s」（#%d）触发 Codex 限额，原因：%s", channelError.ChannelName, channelError.ChannelId, reason)
	if channel.ChannelInfo.IsMultiKey && channelError.UsingKeyIdx >= 0 && channel.Status == common.ChannelStatusEnabled {
		subject = fmt.Sprintf("通道「%s」（#%d）的 Codex Key 已被禁用", channelError.ChannelName, channelError.ChannelId)
		content = fmt.Sprintf("通道「%s」（#%d）的第 %d 个 Key 已因 Codex 限额被禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, channelError.UsingKeyIdx+1, reason)
	}
	NotifyRootUser(formatNotifyType(channelError.ChannelId, channel.Status), subject, content)
	return nil
}

func persistCodexQuotaState(channelID int, keyIndex int, state CodexQuotaState) error {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return err
	}
	states := getCodexQuotaStates(channel)
	states[codexQuotaStateEntryKey(keyIndex)] = state
	return saveCodexQuotaStates(channel, states)
}

func saveCodexQuotaStates(channel *model.Channel, states map[string]CodexQuotaState) error {
	if channel == nil {
		return errors.New("nil channel")
	}
	otherInfo := channel.GetOtherInfo()
	if len(states) == 0 {
		delete(otherInfo, codexQuotaStateOtherInfoKey)
	} else {
		otherInfo[codexQuotaStateOtherInfoKey] = states
	}
	channel.SetOtherInfo(otherInfo)
	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	if common.MemoryCacheEnabled {
		model.CacheUpdateChannel(channel)
	}
	return nil
}

func getCodexQuotaStates(channel *model.Channel) map[string]CodexQuotaState {
	if channel == nil {
		return map[string]CodexQuotaState{}
	}
	otherInfo := channel.GetOtherInfo()
	raw, ok := otherInfo[codexQuotaStateOtherInfoKey]
	if !ok || raw == nil {
		return map[string]CodexQuotaState{}
	}

	encoded, err := common.Marshal(raw)
	if err != nil {
		return map[string]CodexQuotaState{}
	}
	var states map[string]CodexQuotaState
	if err := common.Unmarshal(encoded, &states); err != nil || states == nil {
		return map[string]CodexQuotaState{}
	}
	return states
}

func resolveCodexQuotaStateIndex(channelError types.ChannelError) int {
	if !channelError.IsMultiKey {
		return codexQuotaSingleKeyIndex
	}
	if channelError.UsingKeyIdx >= 0 {
		return channelError.UsingKeyIdx
	}
	return codexQuotaSingleKeyIndex
}

func resolveCodexQuotaUsingKey(channel *model.Channel, keyIndex int) (string, bool) {
	if channel == nil {
		return "", false
	}
	if !channel.ChannelInfo.IsMultiKey || keyIndex == codexQuotaSingleKeyIndex {
		return channel.Key, strings.TrimSpace(channel.Key) != ""
	}
	keys := channel.GetKeys()
	if keyIndex < 0 || keyIndex >= len(keys) {
		return "", false
	}
	return keys[keyIndex], true
}

func isCodexQuotaMessage(message string) bool {
	for _, keyword := range codexQuotaMessageKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func classifyCodexQuotaScopeFromText(message string) string {
	switch {
	case strings.Contains(message, "weekly"), strings.Contains(message, "per week"), strings.Contains(message, "week limit"):
		return codexQuotaScopeWeekly
	case strings.Contains(message, "5 hour"), strings.Contains(message, "5-hour"), strings.Contains(message, "five hour"):
		return codexQuotaScopeFiveHour
	default:
		return codexQuotaScopeUnknown
	}
}

func classifyCodexQuotaScopeByWindowDuration(seconds int64) string {
	switch {
	case seconds <= 0:
		return codexQuotaScopeUnknown
	case seconds >= 24*60*60:
		return codexQuotaScopeWeekly
	default:
		return codexQuotaScopeFiveHour
	}
}

func collectCodexQuotaWindows(meta map[string]interface{}, now time.Time) []codexQuotaWindowCandidate {
	if len(meta) == 0 {
		return nil
	}

	containers := []map[string]interface{}{meta}
	if rateLimit := toStringMap(meta["rate_limit"]); len(rateLimit) > 0 {
		containers = append(containers, rateLimit)
	}

	windows := make([]codexQuotaWindowCandidate, 0, 2)
	for _, container := range containers {
		for _, key := range []string{"primary_window", "secondary_window"} {
			if rawWindow := toStringMap(container[key]); len(rawWindow) > 0 {
				if window := parseCodexQuotaWindow(rawWindow, now); window != nil {
					windows = append(windows, *window)
				}
			}
		}
	}
	return windows
}

func parseCodexQuotaWindow(raw map[string]interface{}, now time.Time) *codexQuotaWindowCandidate {
	if len(raw) == 0 {
		return nil
	}
	resetAt := extractCodexQuotaResetAt(raw, now)
	usedPercent, _ := parseFloat64(raw["used_percent"])
	limitWindowSeconds, _ := parseInt64(raw["limit_window_seconds"])
	return &codexQuotaWindowCandidate{
		Scope:       classifyCodexQuotaScopeByWindowDuration(limitWindowSeconds),
		ResetAt:     resetAt,
		UsedPercent: usedPercent,
	}
}

func selectCodexQuotaWindow(windows []codexQuotaWindowCandidate, preferredScope string) *codexQuotaWindowCandidate {
	if len(windows) == 0 {
		return nil
	}

	filterByScope := func(scope string) []codexQuotaWindowCandidate {
		matches := make([]codexQuotaWindowCandidate, 0, len(windows))
		for _, window := range windows {
			if window.Scope == scope && window.ResetAt > 0 {
				matches = append(matches, window)
			}
		}
		return matches
	}
	pickEarliest := func(candidates []codexQuotaWindowCandidate) *codexQuotaWindowCandidate {
		if len(candidates) == 0 {
			return nil
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].UsedPercent == candidates[j].UsedPercent {
				return candidates[i].ResetAt < candidates[j].ResetAt
			}
			return candidates[i].UsedPercent > candidates[j].UsedPercent
		})
		return &candidates[0]
	}

	if preferredScope != "" && preferredScope != codexQuotaScopeUnknown {
		if picked := pickEarliest(filterByScope(preferredScope)); picked != nil {
			return picked
		}
	}

	exhausted := make([]codexQuotaWindowCandidate, 0, len(windows))
	nearLimit := make([]codexQuotaWindowCandidate, 0, len(windows))
	anyWithReset := make([]codexQuotaWindowCandidate, 0, len(windows))
	for _, window := range windows {
		if window.ResetAt <= 0 {
			continue
		}
		anyWithReset = append(anyWithReset, window)
		if window.UsedPercent >= 100 {
			exhausted = append(exhausted, window)
		} else if window.UsedPercent >= 99.5 {
			nearLimit = append(nearLimit, window)
		}
	}
	if picked := pickEarliest(exhausted); picked != nil {
		return picked
	}
	if picked := pickEarliest(nearLimit); picked != nil {
		return picked
	}
	if preferredScope == codexQuotaScopeUnknown && len(anyWithReset) == 1 {
		return &anyWithReset[0]
	}
	return nil
}

func extractCodexQuotaResetAt(meta map[string]interface{}, now time.Time) int64 {
	if len(meta) == 0 {
		return 0
	}
	for _, key := range []string{"reset_at", "rate_limit_reset"} {
		if resetAt, ok := parseFlexibleResetAt(meta[key], now); ok {
			return resetAt
		}
	}
	for _, key := range []string{"retry_after_seconds", "reset_after_seconds"} {
		if seconds, ok := parseInt64(meta[key]); ok && seconds > 0 {
			return now.Add(time.Duration(seconds) * time.Second).Unix()
		}
	}
	if retryAfter, ok := meta["retry_after"].(string); ok {
		if resetAt, _, ok := parseRetryAfterHeader(retryAfter, now); ok {
			return resetAt
		}
	}
	return 0
}

func parseFlexibleResetAt(value interface{}, now time.Time) (int64, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case string:
		if parsedTime, err := time.Parse(time.RFC3339, strings.TrimSpace(v)); err == nil {
			return parsedTime.Unix(), true
		}
		return parseNumericReset(strings.TrimSpace(v), now)
	case jsonNumber:
		return parseNumericReset(v.String(), now)
	default:
		encoded, err := common.Marshal(v)
		if err != nil {
			return 0, false
		}
		return parseNumericReset(strings.TrimSpace(string(encoded)), now)
	}
}

type jsonNumber interface {
	String() string
}

func parseNumericReset(raw string, now time.Time) (int64, bool) {
	raw = strings.Trim(raw, "\"")
	if raw == "" {
		return 0, false
	}
	num, err := strconv.ParseFloat(raw, 64)
	if err != nil || num <= 0 {
		return 0, false
	}
	switch {
	case num >= 1e12:
		return int64(num / 1000), true
	case num >= 1e9:
		return int64(num), true
	default:
		return now.Add(time.Duration(num) * time.Second).Unix(), true
	}
}

func toStringMap(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	default:
		encoded, err := common.Marshal(typed)
		if err != nil {
			return nil
		}
		var decoded map[string]interface{}
		if err := common.Unmarshal(encoded, &decoded); err != nil {
			return nil
		}
		return decoded
	}
}

func parseInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case string:
		if v == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func parseFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if v == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func codexQuotaStateEntryKey(keyIndex int) string {
	return strconv.Itoa(keyIndex)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
