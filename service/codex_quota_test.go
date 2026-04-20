package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildCodexOAuthKeyForTest(t *testing.T, suffix string) string {
	t.Helper()
	payload, err := common.Marshal(map[string]string{
		"access_token":  "access-" + suffix,
		"account_id":    "account-" + suffix,
		"refresh_token": "refresh-" + suffix,
	})
	require.NoError(t, err)
	return string(payload)
}

func seedCodexChannel(t *testing.T, ch *model.Channel) {
	t.Helper()
	require.NoError(t, model.DB.Create(ch).Error)
}

func TestHandleCodexQuotaChannelErrorSingleKeyPersistsState(t *testing.T) {
	truncate(t)

	autoBan := 1
	channel := &model.Channel{
		Id:      1,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-single",
		Key:     buildCodexOAuthKeyForTest(t, "single"),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
	}
	seedCodexChannel(t, channel)

	err := types.WithOpenAIError(types.OpenAIError{
		Message:  "usage limit reached, try again later",
		Type:     "rate_limit_error",
		Code:     "rate_limit_exceeded",
		Metadata: []byte(`{"reset_at": 2000000000, "limit_window_seconds": 18000}`),
	}, http.StatusTooManyRequests)

	handled := HandleCodexQuotaChannelError(context.Background(), *types.NewChannelError(channel.Id, channel.Type, channel.Name, false, channel.Key, -1, true), err)
	require.True(t, handled)
	assert.Equal(t, types.ErrorCodeChannelCodexQuotaExhausted, err.GetErrorCode())

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	assert.Equal(t, common.ChannelStatusAutoDisabled, loaded.Status)

	states := getCodexQuotaStates(loaded)
	state, ok := states["-1"]
	require.True(t, ok)
	assert.Equal(t, codexQuotaDisableKindQuota, state.DisableKind)
	assert.Equal(t, codexQuotaScopeFiveHour, state.Scope)
	assert.Equal(t, int64(2000000000), state.ResetAt)
	assert.Equal(t, codexQuotaSourceError, state.Source)
	assert.False(t, state.ResetUnknown)
}

func TestHandleCodexQuotaChannelErrorUsesWhamUsageFallback(t *testing.T) {
	truncate(t)

	autoBan := 1
	channel := &model.Channel{
		Id:      2,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-fallback",
		Key:     buildCodexOAuthKeyForTest(t, "fallback"),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
	}
	seedCodexChannel(t, channel)

	originalFetch := fetchCodexWhamUsageForChannelFunc
	fetchCodexWhamUsageForChannelFunc = func(ctx context.Context, channelID int) (int, []byte, error) {
		assert.Equal(t, channel.Id, channelID)
		return http.StatusOK, []byte(`{
			"rate_limit": {
				"primary_window": {"used_percent": 100, "reset_at": 2000000100, "limit_window_seconds": 18000},
				"secondary_window": {"used_percent": 16, "reset_at": 2000000200, "limit_window_seconds": 604800}
			}
		}`), nil
	}
	t.Cleanup(func() {
		fetchCodexWhamUsageForChannelFunc = originalFetch
	})

	err := types.WithOpenAIError(types.OpenAIError{
		Message: "usage limit reached, try again later",
		Type:    "rate_limit_error",
		Code:    "rate_limit_exceeded",
	}, http.StatusTooManyRequests)

	handled := HandleCodexQuotaChannelError(context.Background(), *types.NewChannelError(channel.Id, channel.Type, channel.Name, false, channel.Key, -1, true), err)
	require.True(t, handled)

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	states := getCodexQuotaStates(loaded)
	state, ok := states["-1"]
	require.True(t, ok)
	assert.Equal(t, codexQuotaSourceUsage, state.Source)
	assert.Equal(t, int64(2000000100), state.ResetAt)
	assert.Equal(t, codexQuotaScopeFiveHour, state.Scope)
}

func TestHandleCodexQuotaChannelErrorMultiKeyDisablesOnlyCurrentKey(t *testing.T) {
	truncate(t)

	autoBan := 1
	keyA := buildCodexOAuthKeyForTest(t, "multi-a")
	keyB := buildCodexOAuthKeyForTest(t, "multi-b")
	channel := &model.Channel{
		Id:      3,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-multi",
		Key:     fmt.Sprintf("%s\n%s", keyA, keyB),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:   true,
			MultiKeySize: 2,
		},
	}
	seedCodexChannel(t, channel)

	err := types.WithOpenAIError(types.OpenAIError{
		Message:  "weekly usage limit reached",
		Type:     "rate_limit_error",
		Code:     "rate_limit_exceeded",
		Metadata: []byte(`{"reset_at": 2000000300, "limit_window_seconds": 604800}`),
	}, http.StatusTooManyRequests)

	handled := HandleCodexQuotaChannelError(context.Background(), *types.NewChannelError(channel.Id, channel.Type, channel.Name, true, keyB, 1, true), err)
	require.True(t, handled)

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	assert.Equal(t, common.ChannelStatusEnabled, loaded.Status)
	require.Contains(t, loaded.ChannelInfo.MultiKeyStatusList, 1)
	assert.Equal(t, common.ChannelStatusAutoDisabled, loaded.ChannelInfo.MultiKeyStatusList[1])
	_, firstKeyDisabled := loaded.ChannelInfo.MultiKeyStatusList[0]
	assert.False(t, firstKeyDisabled)

	states := getCodexQuotaStates(loaded)
	state, ok := states["1"]
	require.True(t, ok)
	assert.Equal(t, codexQuotaScopeWeekly, state.Scope)
	assert.Equal(t, int64(2000000300), state.ResetAt)
}

func TestRunCodexQuotaAutoReenablePassRecoversDueStates(t *testing.T) {
	truncate(t)

	autoBan := 1
	channel := &model.Channel{
		Id:      4,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-recover",
		Key:     buildCodexOAuthKeyForTest(t, "recover"),
		Status:  common.ChannelStatusAutoDisabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
	}
	channel.SetOtherInfo(map[string]interface{}{
		codexQuotaStateOtherInfoKey: map[string]CodexQuotaState{
			"-1": {
				DisableKind: codexQuotaDisableKindQuota,
				Scope:       codexQuotaScopeFiveHour,
				ResetAt:     common.GetTimestamp() - 10,
				Source:      codexQuotaSourceError,
				DisabledAt:  common.GetTimestamp() - 30,
			},
		},
	})
	seedCodexChannel(t, channel)

	require.NoError(t, runCodexQuotaAutoReenablePass(context.Background(), common.GetTimestamp()))

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	assert.Equal(t, common.ChannelStatusEnabled, loaded.Status)
	assert.Empty(t, getCodexQuotaStates(loaded))
}

func TestRunCodexQuotaAutoReenablePassRecoversMultiKeyEntry(t *testing.T) {
	truncate(t)

	autoBan := 1
	keyA := buildCodexOAuthKeyForTest(t, "recover-a")
	keyB := buildCodexOAuthKeyForTest(t, "recover-b")
	channel := &model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "codex-recover-multi",
		Key:     fmt.Sprintf("%s\n%s", keyA, keyB),
		Status:  common.ChannelStatusEnabled,
		AutoBan: &autoBan,
		Models:  "gpt-5.4",
		Group:   "default",
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       2,
			MultiKeyStatusList: map[int]int{1: common.ChannelStatusAutoDisabled},
		},
	}
	channel.SetOtherInfo(map[string]interface{}{
		codexQuotaStateOtherInfoKey: map[string]CodexQuotaState{
			"1": {
				DisableKind: codexQuotaDisableKindQuota,
				Scope:       codexQuotaScopeWeekly,
				ResetAt:     common.GetTimestamp() - 10,
				Source:      codexQuotaSourceError,
				DisabledAt:  common.GetTimestamp() - 30,
			},
		},
	})
	seedCodexChannel(t, channel)

	require.NoError(t, runCodexQuotaAutoReenablePass(context.Background(), common.GetTimestamp()))

	loaded, loadErr := model.GetChannelById(channel.Id, true)
	require.NoError(t, loadErr)
	assert.Equal(t, common.ChannelStatusEnabled, loaded.Status)
	assert.NotContains(t, loaded.ChannelInfo.MultiKeyStatusList, 1)
	assert.Empty(t, getCodexQuotaStates(loaded))
}
