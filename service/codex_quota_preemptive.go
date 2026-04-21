package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	codexQuotaUsageSnapshotOtherInfoKey = "codex_usage_snapshot"

	codexQuotaPreemptiveTickInterval             = time.Minute
	codexQuotaPreemptiveDefaultRefreshInterval   = 10 * time.Minute
	codexQuotaPreemptiveNearLimitRefreshInterval = 3 * time.Minute
	codexQuotaPreemptiveFetchTimeout             = 15 * time.Second
	codexQuotaPreemptiveBatchSize                = 200
	codexQuotaPreemptiveJitterCap                = 30 * time.Second

	codexQuotaNearLimitPercent        = 90.0
	codexQuotaFiveHourCooldownPercent = 95.0
	codexQuotaWeeklyCooldownPercent   = 98.0

	codexQuotaUsageSnapshotSourceUsageProbe = "usage_probe"
)

var (
	codexQuotaPreemptiveRefreshOnce    sync.Once
	codexQuotaPreemptiveRefreshRunning atomic.Bool
	codexQuotaPreemptiveChannelStates  sync.Map

	codexQuotaPreemptiveNowFunc    = time.Now
	codexQuotaPreemptiveJitterFunc = func(interval time.Duration) time.Duration {
		if interval <= 0 {
			return 0
		}
		maxJitter := interval / 10
		if maxJitter <= 0 || maxJitter > codexQuotaPreemptiveJitterCap {
			maxJitter = codexQuotaPreemptiveJitterCap
		}
		if maxJitter <= 0 {
			return 0
		}
		maxSeconds := int(maxJitter / time.Second)
		if maxSeconds <= 0 {
			return 0
		}
		return time.Duration(common.GetRandomInt(maxSeconds+1)) * time.Second
	}
)

type CodexUsageSnapshot struct {
	LastCheckedAt       int64   `json:"last_checked_at,omitempty"`
	NextCheckAt         int64   `json:"next_check_at,omitempty"`
	FiveHourUsedPercent float64 `json:"five_hour_used_percent,omitempty"`
	FiveHourResetAt     int64   `json:"five_hour_reset_at,omitempty"`
	WeeklyUsedPercent   float64 `json:"weekly_used_percent,omitempty"`
	WeeklyResetAt       int64   `json:"weekly_reset_at,omitempty"`
	Source              string  `json:"source,omitempty"`
	LastError           string  `json:"last_error,omitempty"`
}

type codexQuotaWindowState struct {
	UsedPercent float64
	ResetAt     int64
}

func StartCodexQuotaPreemptiveRefreshTask() {
	codexQuotaPreemptiveRefreshOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"codex quota preemptive refresh task started: tick=%s default_refresh=%s near_limit_refresh=%s thresholds={five_hour:%.1f%%,weekly:%.1f%%}",
				codexQuotaPreemptiveTickInterval,
				codexQuotaPreemptiveDefaultRefreshInterval,
				codexQuotaPreemptiveNearLimitRefreshInterval,
				codexQuotaFiveHourCooldownPercent,
				codexQuotaWeeklyCooldownPercent,
			))

			ticker := time.NewTicker(codexQuotaPreemptiveTickInterval)
			defer ticker.Stop()

			runCodexQuotaPreemptiveRefreshOnce()
			for range ticker.C {
				runCodexQuotaPreemptiveRefreshOnce()
			}
		})
	})
}

func runCodexQuotaPreemptiveRefreshOnce() {
	if !codexQuotaPreemptiveRefreshRunning.CompareAndSwap(false, true) {
		return
	}
	defer codexQuotaPreemptiveRefreshRunning.Store(false)

	ctx := context.Background()
	now := codexQuotaPreemptiveNowFunc()

	var scanned int
	var refreshed int

	offset := 0
	for {
		var channels []*model.Channel
		err := model.DB.
			Select("id", "name", "type", "status", "channel_info", "other_info").
			Where("type = ? AND status = ?", constant.ChannelTypeCodex, common.ChannelStatusEnabled).
			Order("id asc").
			Limit(codexQuotaPreemptiveBatchSize).
			Offset(offset).
			Find(&channels).Error
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("codex quota preemptive refresh: query channels failed: %v", err))
			return
		}
		if len(channels) == 0 {
			break
		}
		offset += codexQuotaPreemptiveBatchSize

		for _, channel := range channels {
			if channel == nil {
				continue
			}
			scanned++
			if !shouldRunCodexQuotaPreemptiveRefresh(channel, now.Unix()) {
				continue
			}
			unlock := tryLockCodexQuotaPreemptiveChannel(channel.Id)
			if unlock == nil {
				continue
			}
			err := refreshCodexQuotaPreemptiveSnapshot(ctx, channel, now)
			unlock()
			if err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("codex quota preemptive refresh: channel_id=%d name=%s failed: %v", channel.Id, channel.Name, err))
				continue
			}
			refreshed++
		}
	}

	if common.DebugEnabled {
		logger.LogDebug(ctx, "codex quota preemptive refresh: scanned=%d refreshed=%d", scanned, refreshed)
	}
}

func shouldRunCodexQuotaPreemptiveRefresh(channel *model.Channel, now int64) bool {
	if channel == nil || channel.Type != constant.ChannelTypeCodex || channel.Status != common.ChannelStatusEnabled {
		return false
	}
	if channel.ChannelInfo.IsMultiKey {
		return false
	}
	snapshot := getCodexUsageSnapshot(channel)
	return snapshot.NextCheckAt == 0 || snapshot.NextCheckAt <= now
}

func tryLockCodexQuotaPreemptiveChannel(channelID int) func() {
	if channelID <= 0 {
		return nil
	}
	stateAny, _ := codexQuotaPreemptiveChannelStates.LoadOrStore(channelID, &atomic.Bool{})
	state, ok := stateAny.(*atomic.Bool)
	if !ok {
		return nil
	}
	if !state.CompareAndSwap(false, true) {
		return nil
	}
	return func() {
		state.Store(false)
	}
}

func refreshCodexQuotaPreemptiveSnapshot(ctx context.Context, channel *model.Channel, now time.Time) error {
	if channel == nil {
		return errors.New("nil channel")
	}
	if channel.ChannelInfo.IsMultiKey {
		return nil
	}

	refreshCtx, cancel := context.WithTimeout(ctx, codexQuotaPreemptiveFetchTimeout)
	defer cancel()

	statusCode, body, err := fetchCodexWhamUsageForChannelFunc(refreshCtx, channel.Id)
	if err != nil {
		return persistCodexQuotaPreemptiveFailure(channel, now, err.Error())
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return persistCodexQuotaPreemptiveFailure(channel, now, fmt.Sprintf("codex usage upstream status=%d", statusCode))
	}

	payload := make(map[string]interface{})
	if err := common.Unmarshal(body, &payload); err != nil {
		return persistCodexQuotaPreemptiveFailure(channel, now, fmt.Sprintf("invalid codex usage payload: %v", err))
	}

	windows := collectCodexQuotaWindows(payload, now)
	snapshot := buildCodexUsageSnapshotFromWindows(windows, now)
	if snapshot.Source == "" {
		snapshot.Source = codexQuotaUsageSnapshotSourceUsageProbe
	}
	setCodexUsageSnapshotSchedule(&snapshot, now)

	cooldownState := buildCodexQuotaPreemptiveCooldownState(snapshot, now)
	return persistCodexQuotaPreemptiveSnapshot(channel, snapshot, cooldownState)
}

func buildCodexUsageSnapshotFromWindows(windows []codexQuotaWindowCandidate, now time.Time) CodexUsageSnapshot {
	fiveHourWindow, weeklyWindow := summarizeCodexQuotaWindows(windows)
	return CodexUsageSnapshot{
		LastCheckedAt:       now.Unix(),
		FiveHourUsedPercent: fiveHourWindow.UsedPercent,
		FiveHourResetAt:     fiveHourWindow.ResetAt,
		WeeklyUsedPercent:   weeklyWindow.UsedPercent,
		WeeklyResetAt:       weeklyWindow.ResetAt,
		Source:              codexQuotaUsageSnapshotSourceUsageProbe,
	}
}

func summarizeCodexQuotaWindows(windows []codexQuotaWindowCandidate) (codexQuotaWindowState, codexQuotaWindowState) {
	var fiveHourWindow codexQuotaWindowState
	var weeklyWindow codexQuotaWindowState
	for _, window := range windows {
		switch window.Scope {
		case codexQuotaScopeFiveHour:
			if shouldReplaceCodexQuotaWindow(fiveHourWindow, window) {
				fiveHourWindow = codexQuotaWindowState{UsedPercent: window.UsedPercent, ResetAt: window.ResetAt}
			}
		case codexQuotaScopeWeekly:
			if shouldReplaceCodexQuotaWindow(weeklyWindow, window) {
				weeklyWindow = codexQuotaWindowState{UsedPercent: window.UsedPercent, ResetAt: window.ResetAt}
			}
		}
	}
	return fiveHourWindow, weeklyWindow
}

func shouldReplaceCodexQuotaWindow(current codexQuotaWindowState, candidate codexQuotaWindowCandidate) bool {
	if candidate.ResetAt <= 0 {
		return false
	}
	if current.ResetAt == 0 {
		return true
	}
	if candidate.UsedPercent == current.UsedPercent {
		return candidate.ResetAt > current.ResetAt
	}
	return candidate.UsedPercent > current.UsedPercent
}

func setCodexUsageSnapshotSchedule(snapshot *CodexUsageSnapshot, now time.Time) {
	if snapshot == nil {
		return
	}
	interval := codexQuotaPreemptiveDefaultRefreshInterval
	if snapshot.FiveHourUsedPercent >= codexQuotaNearLimitPercent || snapshot.WeeklyUsedPercent >= codexQuotaNearLimitPercent {
		interval = codexQuotaPreemptiveNearLimitRefreshInterval
	}
	nextCheckAt := now.Add(interval + codexQuotaPreemptiveJitterFunc(interval)).Unix()
	if snapshot.FiveHourResetAt > 0 && snapshot.FiveHourResetAt < nextCheckAt {
		nextCheckAt = snapshot.FiveHourResetAt
	}
	if snapshot.WeeklyResetAt > 0 && snapshot.WeeklyResetAt < nextCheckAt {
		nextCheckAt = snapshot.WeeklyResetAt
	}
	if nextCheckAt <= now.Unix() {
		nextCheckAt = now.Add(interval).Unix()
	}
	snapshot.NextCheckAt = nextCheckAt
}

func buildCodexQuotaPreemptiveCooldownState(snapshot CodexUsageSnapshot, now time.Time) *model.RoutingCooldownState {
	qualified := make([]codexQuotaWindowCandidate, 0, 2)
	if snapshot.FiveHourUsedPercent >= codexQuotaFiveHourCooldownPercent && snapshot.FiveHourResetAt > now.Unix() {
		qualified = append(qualified, codexQuotaWindowCandidate{
			Scope:       codexQuotaScopeFiveHour,
			ResetAt:     snapshot.FiveHourResetAt,
			UsedPercent: snapshot.FiveHourUsedPercent,
		})
	}
	if snapshot.WeeklyUsedPercent >= codexQuotaWeeklyCooldownPercent && snapshot.WeeklyResetAt > now.Unix() {
		qualified = append(qualified, codexQuotaWindowCandidate{
			Scope:       codexQuotaScopeWeekly,
			ResetAt:     snapshot.WeeklyResetAt,
			UsedPercent: snapshot.WeeklyUsedPercent,
		})
	}
	if len(qualified) == 0 {
		return nil
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].ResetAt > qualified[j].ResetAt
	})

	return &model.RoutingCooldownState{
		Kind:      channelRoutingCooldownKindQuota,
		Reason:    buildCodexQuotaPreemptiveReason(qualified),
		ResetAt:   qualified[0].ResetAt,
		Source:    channelRoutingCooldownSourceUsageProbe,
		KeyIndex:  -1,
		CreatedAt: now.Unix(),
	}
}

func buildCodexQuotaPreemptiveReason(windows []codexQuotaWindowCandidate) string {
	if len(windows) == 0 {
		return "Codex preemptive quota cooldown"
	}
	parts := make([]string, 0, len(windows))
	for _, window := range windows {
		switch window.Scope {
		case codexQuotaScopeFiveHour:
			parts = append(parts, fmt.Sprintf("5h=%.1f%%", window.UsedPercent))
		case codexQuotaScopeWeekly:
			parts = append(parts, fmt.Sprintf("weekly=%.1f%%", window.UsedPercent))
		}
	}
	if len(parts) == 0 {
		return "Codex preemptive quota cooldown"
	}
	return fmt.Sprintf("Codex preemptive quota cooldown (%s)", strings.Join(parts, ", "))
}

func persistCodexQuotaPreemptiveFailure(channel *model.Channel, now time.Time, lastError string) error {
	if channel == nil {
		return errors.New("nil channel")
	}
	snapshot := getCodexUsageSnapshot(channel)
	snapshot.LastCheckedAt = now.Unix()
	snapshot.NextCheckAt = now.Add(codexQuotaPreemptiveDefaultRefreshInterval + codexQuotaPreemptiveJitterFunc(codexQuotaPreemptiveDefaultRefreshInterval)).Unix()
	snapshot.Source = codexQuotaUsageSnapshotSourceUsageProbe
	snapshot.LastError = strings.TrimSpace(lastError)
	return persistCodexQuotaPreemptiveSnapshot(channel, snapshot, nil)
}

func persistCodexQuotaPreemptiveSnapshot(channel *model.Channel, snapshot CodexUsageSnapshot, cooldownState *model.RoutingCooldownState) error {
	if channel == nil {
		return errors.New("nil channel")
	}
	channel.SetOtherInfo(mergeCodexUsageSnapshotOtherInfo(channel.GetOtherInfo(), snapshot))
	if cooldownState != nil {
		states := channel.GetRoutingCooldownStates()
		entryKey := "-1"
		if existing, ok := states[entryKey]; ok && existing.ResetAt > cooldownState.ResetAt {
			cooldownState = &existing
		} else {
			states[entryKey] = *cooldownState
		}
		channel.SetRoutingCooldownStates(states)
	}
	if err := model.DB.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("other_info", channel.OtherInfo).Error; err != nil {
		return err
	}
	if common.MemoryCacheEnabled {
		if cacheChannel, err := model.CacheGetChannel(channel.Id); err == nil && cacheChannel != nil {
			cacheChannel.OtherInfo = channel.OtherInfo
		}
	}
	return nil
}

func mergeCodexUsageSnapshotOtherInfo(otherInfo map[string]interface{}, snapshot CodexUsageSnapshot) map[string]interface{} {
	if otherInfo == nil {
		otherInfo = make(map[string]interface{})
	}
	otherInfo[codexQuotaUsageSnapshotOtherInfoKey] = snapshot
	return otherInfo
}

func getCodexUsageSnapshot(channel *model.Channel) CodexUsageSnapshot {
	if channel == nil {
		return CodexUsageSnapshot{}
	}
	otherInfo := channel.GetOtherInfo()
	raw, ok := otherInfo[codexQuotaUsageSnapshotOtherInfoKey]
	if !ok || raw == nil {
		return CodexUsageSnapshot{}
	}
	encoded, err := common.Marshal(raw)
	if err != nil {
		return CodexUsageSnapshot{}
	}
	var snapshot CodexUsageSnapshot
	if err := common.Unmarshal(encoded, &snapshot); err != nil {
		return CodexUsageSnapshot{}
	}
	return snapshot
}
