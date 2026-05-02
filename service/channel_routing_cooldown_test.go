package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedChannelForRoutingTest(t *testing.T, channel *model.Channel, priority int64) {
	t.Helper()
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     channel.Group,
		Model:     channel.Models,
		ChannelId: channel.Id,
		Enabled:   true,
		Priority:  common.GetPointer(priority),
		Weight:    0,
	}).Error)
}

func TestGetRandomSatisfiedChannelSkipsRoutingCooldownAndExcludedChannel(t *testing.T) {
	truncate(t)
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		model.InitChannelCache()
	})

	priorityHigh := int64(10)
	priorityLow := int64(5)
	now := common.GetTimestamp()

	channelA := &model.Channel{
		Id:       101,
		Type:     constant.ChannelTypeOpenAI,
		Name:     "channel-a",
		Key:      "sk-a",
		Status:   common.ChannelStatusEnabled,
		Models:   "gpt-5",
		Group:    "default",
		Priority: &priorityHigh,
	}
	channelA.SetRoutingCooldownStates(map[string]model.RoutingCooldownState{
		"-1": {
			Kind:      channelRoutingCooldownKindQuota,
			Reason:    "quota hit",
			ResetAt:   now + 600,
			Source:    channelRoutingCooldownSourceFallback,
			CreatedAt: now,
		},
	})

	channelB := &model.Channel{
		Id:       102,
		Type:     constant.ChannelTypeOpenAI,
		Name:     "channel-b",
		Key:      "sk-b",
		Status:   common.ChannelStatusEnabled,
		Models:   "gpt-5",
		Group:    "default",
		Priority: &priorityHigh,
	}

	channelC := &model.Channel{
		Id:       103,
		Type:     constant.ChannelTypeOpenAI,
		Name:     "channel-c",
		Key:      "sk-c",
		Status:   common.ChannelStatusEnabled,
		Models:   "gpt-5",
		Group:    "default",
		Priority: &priorityLow,
	}

	seedChannelForRoutingTest(t, channelA, priorityHigh)
	seedChannelForRoutingTest(t, channelB, priorityHigh)
	seedChannelForRoutingTest(t, channelC, priorityLow)
	model.InitChannelCache()

	selected, err := model.GetRandomSatisfiedChannel("default", "gpt-5", 0, nil)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channelB.Id, selected.Id)

	selected, err = model.GetRandomSatisfiedChannel("default", "gpt-5", 0, map[int]struct{}{
		channelB.Id: {},
	})
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channelC.Id, selected.Id)
}

func TestGetNextEnabledKeySkipsRoutingCooledPollingKey(t *testing.T) {
	now := common.GetTimestamp()
	channel := &model.Channel{
		Id:     201,
		Key:    "key-a\nkey-b",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:           true,
			MultiKeySize:         2,
			MultiKeyMode:         constant.MultiKeyModePolling,
			MultiKeyPollingIndex: 1,
		},
	}
	channel.SetRoutingCooldownStates(map[string]model.RoutingCooldownState{
		"1": {
			Kind:      channelRoutingCooldownKindQuota,
			ResetAt:   now + 600,
			Source:    channelRoutingCooldownSourceFallback,
			KeyIndex:  1,
			CreatedAt: now,
		},
	})
	require.NoError(t, model.DB.Create(channel).Error)

	key, idx, apiErr := channel.GetNextEnabledKey()
	require.Nil(t, apiErr)
	assert.Equal(t, "key-a", key)
	assert.Equal(t, 0, idx)
}

func TestShouldExcludeChannelAfterFailureAvoidsFailedMultiKeyChannel(t *testing.T) {
	ctx := newCodexQuotaTestContext()
	setRoutingCooldownAdminInfo(ctx, model.RoutingCooldownState{
		Kind:      channelRoutingCooldownKindQuota,
		ResetAt:   common.GetTimestamp() + 600,
		Source:    channelRoutingCooldownSourceFallback,
		KeyIndex:  1,
		CreatedAt: common.GetTimestamp(),
	})

	channelError := *types.NewChannelError(301, constant.ChannelTypeCodex, "codex-multi", true, "key-b", 1, true)
	assert.True(t, ShouldExcludeChannelAfterFailure(channelError, ctx))
}

func TestClassifyGenericRoutingCooldownCoolsChineseQuota403UntilNextDay(t *testing.T) {
	err := types.NewOpenAIError(
		assert.AnError,
		types.ErrorCodeBadResponseStatusCode,
		http.StatusForbidden,
	)
	err.SetMessage("用户额度不足, 剩余额度: ¥0.000000")

	state, ok := classifyGenericRoutingCooldown(constant.ChannelTypeOpenAI, err)
	require.True(t, ok)
	assert.Equal(t, channelRoutingCooldownKindQuota, state.Kind)
	assert.Equal(t, channelRoutingCooldownSourceDailyQuota, state.Source)
	assert.Contains(t, state.Reason, "用户额度不足")

	now := time.Now()
	maxReset := now.Add(24*time.Hour + dailyQuotaResetDelay).Unix()
	assert.Greater(t, state.ResetAt, now.Unix())
	assert.Less(t, state.ResetAt, maxReset)
}

func TestClassifyGenericRoutingCooldownCoolsDailyLimit429UntilNextDay(t *testing.T) {
	err := types.NewOpenAIError(
		assert.AnError,
		types.ErrorCodeBadResponseStatusCode,
		http.StatusTooManyRequests,
	)
	err.SetMessage(`error: code=429 reason="DAILY_LIMIT_EXCEEDED" message="daily usage limit exceeded" metadata=map[]`)

	state, ok := classifyGenericRoutingCooldown(constant.ChannelTypeOpenAI, err)
	require.True(t, ok)
	assert.Equal(t, channelRoutingCooldownKindQuota, state.Kind)
	assert.Equal(t, channelRoutingCooldownSourceDailyQuota, state.Source)
	assert.Contains(t, state.Reason, "DAILY_LIMIT_EXCEEDED")

	now := time.Now()
	maxReset := now.Add(24*time.Hour + dailyQuotaResetDelay).Unix()
	assert.Greater(t, state.ResetAt, now.Unix()+int64(defaultChannelRoutingCooldown/time.Second))
	assert.Less(t, state.ResetAt, maxReset)
}

func TestClassifyGenericRoutingCooldownDoesNotCoolGenericForbidden(t *testing.T) {
	err := types.NewOpenAIError(
		assert.AnError,
		types.ErrorCodeBadResponseStatusCode,
		http.StatusForbidden,
	)
	err.SetMessage("permission denied")

	_, ok := classifyGenericRoutingCooldown(constant.ChannelTypeOpenAI, err)
	assert.False(t, ok)
}

func TestClassifyGenericRoutingCooldownDoesNotCoolCodexQuota403(t *testing.T) {
	err := types.NewOpenAIError(
		assert.AnError,
		types.ErrorCodeBadResponseStatusCode,
		http.StatusForbidden,
	)
	err.SetMessage("用户额度不足, 剩余额度: ¥0.000000")

	_, ok := classifyGenericRoutingCooldown(constant.ChannelTypeCodex, err)
	assert.False(t, ok)
}
