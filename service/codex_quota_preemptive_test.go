package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withCodexQuotaPreemptiveTestHooks(t *testing.T, now time.Time) {
	t.Helper()
	originalNowFunc := codexQuotaPreemptiveNowFunc
	originalJitterFunc := codexQuotaPreemptiveJitterFunc
	originalFetch := fetchCodexWhamUsageForChannelFunc
	codexQuotaPreemptiveNowFunc = func() time.Time { return now }
	codexQuotaPreemptiveJitterFunc = func(time.Duration) time.Duration { return 0 }
	t.Cleanup(func() {
		codexQuotaPreemptiveNowFunc = originalNowFunc
		codexQuotaPreemptiveJitterFunc = originalJitterFunc
		fetchCodexWhamUsageForChannelFunc = originalFetch
	})
}

func seedSingleKeyCodexChannelForPreemptiveTest(t *testing.T, id int, name string) *model.Channel {
	t.Helper()
	autoBan := 1
	channel := &model.Channel{
		Id:      id,
		Type:    constant.ChannelTypeCodex,
		Name:    name,
		Key:     buildCodexOAuthKeyForTest(t, name),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	return channel
}

func codexUsagePayload(t *testing.T, primaryUsed float64, primaryReset int64, secondaryUsed float64, secondaryReset int64) []byte {
	t.Helper()
	body, err := common.Marshal(map[string]interface{}{
		"rate_limit": map[string]interface{}{
			"primary_window": map[string]interface{}{
				"used_percent":         primaryUsed,
				"reset_at":             primaryReset,
				"limit_window_seconds": 18000,
			},
			"secondary_window": map[string]interface{}{
				"used_percent":         secondaryUsed,
				"reset_at":             secondaryReset,
				"limit_window_seconds": 604800,
			},
		},
	})
	require.NoError(t, err)
	return body
}

func TestRefreshCodexQuotaPreemptiveSnapshotAppliesFiveHourCooldown(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+600, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 11, "codex-five-hour")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		assert.Equal(t, channel.Id, channelID)
		return http.StatusOK, codexUsagePayload(t, 95, now.Add(2*time.Hour).Unix(), 45, now.Add(5*24*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now))

	loaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)

	snapshot := getCodexUsageSnapshot(loaded)
	assert.Equal(t, now.Unix(), snapshot.LastCheckedAt)
	assert.Equal(t, now.Add(3*time.Minute).Unix(), snapshot.NextCheckAt)
	assert.Equal(t, 95.0, snapshot.FiveHourUsedPercent)
	assert.Equal(t, now.Add(2*time.Hour).Unix(), snapshot.FiveHourResetAt)
	assert.Equal(t, 45.0, snapshot.WeeklyUsedPercent)
	assert.Equal(t, codexQuotaUsageSnapshotSourceUsageProbe, snapshot.Source)
	assert.Empty(t, snapshot.LastError)

	states := loaded.GetRoutingCooldownStates()
	state, ok := states["-1"]
	require.True(t, ok)
	assert.Equal(t, channelRoutingCooldownKindQuota, state.Kind)
	assert.Equal(t, now.Add(2*time.Hour).Unix(), state.ResetAt)
	assert.Equal(t, channelRoutingCooldownSourceUsageProbe, state.Source)
	assert.Contains(t, state.Reason, "5h=95.0%")
}

func TestRefreshCodexQuotaPreemptiveSnapshotAppliesWeeklyCooldown(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+700, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 12, "codex-weekly")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		return http.StatusOK, codexUsagePayload(t, 50, now.Add(90*time.Minute).Unix(), 98, now.Add(24*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now))

	loaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)
	states := loaded.GetRoutingCooldownStates()
	state, ok := states["-1"]
	require.True(t, ok)
	assert.Equal(t, now.Add(24*time.Hour).Unix(), state.ResetAt)
	assert.Contains(t, state.Reason, "weekly=98.0%")
}

func TestRefreshCodexQuotaPreemptiveSnapshotUsesLaterResetWhenBothThresholdsHit(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+800, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 13, "codex-both")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		return http.StatusOK, codexUsagePayload(t, 96, now.Add(2*time.Hour).Unix(), 99, now.Add(36*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now))

	loaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)
	state := loaded.GetRoutingCooldownStates()["-1"]
	assert.Equal(t, now.Add(36*time.Hour).Unix(), state.ResetAt)
	assert.Contains(t, state.Reason, "5h=96.0%")
	assert.Contains(t, state.Reason, "weekly=99.0%")
}

func TestRefreshCodexQuotaPreemptiveSnapshotSpeedsUpNearLimitWithoutCooldown(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+900, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 14, "codex-near-limit")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		return http.StatusOK, codexUsagePayload(t, 92, now.Add(90*time.Minute).Unix(), 80, now.Add(3*24*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now))

	loaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)
	snapshot := getCodexUsageSnapshot(loaded)
	assert.Equal(t, now.Add(3*time.Minute).Unix(), snapshot.NextCheckAt)
	assert.Empty(t, loaded.GetRoutingCooldownStates())
}

func TestRefreshCodexQuotaPreemptiveSnapshotUsesDefaultIntervalBelowNearLimit(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+1000, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 15, "codex-normal")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		return http.StatusOK, codexUsagePayload(t, 70, now.Add(2*time.Hour).Unix(), 50, now.Add(2*24*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now))

	loaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)
	snapshot := getCodexUsageSnapshot(loaded)
	assert.Equal(t, now.Add(10*time.Minute).Unix(), snapshot.NextCheckAt)
	assert.Empty(t, loaded.GetRoutingCooldownStates())
}

func TestRefreshCodexQuotaPreemptiveSnapshotFailureDoesNotApplyCooldown(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+1100, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	channel := seedSingleKeyCodexChannelForPreemptiveTest(t, 16, "codex-failure")
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		return http.StatusTooManyRequests, nil, nil
	}

	err := refreshCodexQuotaPreemptiveSnapshot(context.Background(), channel, now)
	require.NoError(t, err)

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	snapshot := getCodexUsageSnapshot(loaded)
	assert.Equal(t, now.Unix(), snapshot.LastCheckedAt)
	assert.Equal(t, now.Add(10*time.Minute).Unix(), snapshot.NextCheckAt)
	assert.Contains(t, snapshot.LastError, "upstream status=429")
	assert.Empty(t, loaded.GetRoutingCooldownStates())
}

func TestRunCodexQuotaPreemptiveRefreshOnceSkipsMultiKeyChannels(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+1200, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	singleKey := seedSingleKeyCodexChannelForPreemptiveTest(t, 17, "codex-single")
	autoBan := 1
	multiKey := &model.Channel{
		Id:      18,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-multi",
		Key:     fmt.Sprintf("%s\n%s", buildCodexOAuthKeyForTest(t, "mk-a"), buildCodexOAuthKeyForTest(t, "mk-b")),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:   true,
			MultiKeySize: 2,
		},
	}
	require.NoError(t, model.DB.Create(multiKey).Error)

	var fetched []int
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		fetched = append(fetched, channelID)
		return http.StatusOK, codexUsagePayload(t, 80, now.Add(2*time.Hour).Unix(), 50, now.Add(24*time.Hour).Unix()), nil
	}

	runCodexQuotaPreemptiveRefreshOnce()

	require.Equal(t, []int{singleKey.Id}, fetched)
}

func TestCodexPreemptiveCooldownMakesChannelUnavailableForSelection(t *testing.T) {
	truncate(t)
	now := time.Unix(common.GetTimestamp()+1300, 0)
	withCodexQuotaPreemptiveTestHooks(t, now)

	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		model.InitChannelCache()
	})

	priority := int64(10)
	channelA := &model.Channel{
		Id:       19,
		Type:     constant.ChannelTypeCodex,
		Name:     "codex-a",
		Key:      buildCodexOAuthKeyForTest(t, "sel-a"),
		Status:   common.ChannelStatusEnabled,
		Models:   "gpt-5.4",
		Group:    "default",
		Priority: &priority,
	}
	channelB := &model.Channel{
		Id:       20,
		Type:     constant.ChannelTypeCodex,
		Name:     "codex-b",
		Key:      buildCodexOAuthKeyForTest(t, "sel-b"),
		Status:   common.ChannelStatusEnabled,
		Models:   "gpt-5.4",
		Group:    "default",
		Priority: &priority,
	}
	require.NoError(t, model.DB.Create(channelA).Error)
	require.NoError(t, model.DB.Create(channelB).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-5.4", ChannelId: channelA.Id, Enabled: true, Priority: &priority, Weight: 0}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-5.4", ChannelId: channelB.Id, Enabled: true, Priority: &priority, Weight: 0}).Error)
	model.InitChannelCache()

	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		if channelID == channelA.Id {
			return http.StatusOK, codexUsagePayload(t, 96, now.Add(2*time.Hour).Unix(), 50, now.Add(24*time.Hour).Unix()), nil
		}
		return http.StatusOK, codexUsagePayload(t, 40, now.Add(2*time.Hour).Unix(), 20, now.Add(24*time.Hour).Unix()), nil
	}

	require.NoError(t, refreshCodexQuotaPreemptiveSnapshot(context.Background(), channelA, now))

	selected, err := model.GetRandomSatisfiedChannel("default", "gpt-5.4", 0, nil)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channelB.Id, selected.Id)
}
