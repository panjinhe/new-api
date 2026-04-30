package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResetRuntimeStateStripsRuntimeMetadata(t *testing.T) {
	channel := &Channel{
		TestTime:           123,
		ResponseTime:       456,
		Balance:            7.89,
		BalanceUpdatedTime: 789,
		UsedQuota:          999,
		OtherInfo:          `{"source":"ionet","deployment_id":"dep-1","routing_cooldown":{"-1":{"reset_at":123}},"codex_usage_snapshot":{"weekly_used_percent":100},"codex_quota_state":{"-1":{"reset_at":456}},"status_reason":"quota","status_time":111}`,
		ChannelInfo: ChannelInfo{
			IsMultiKey:             true,
			MultiKeySize:           2,
			MultiKeyStatusList:     map[int]int{0: 2},
			MultiKeyDisabledReason: map[int]string{0: "quota"},
			MultiKeyDisabledTime:   map[int]int64{0: 123},
			MultiKeyPollingIndex:   1,
		},
		RoutingCooldownActive:  true,
		RoutingCooldownResetAt: 999,
		RoutingCooldownReason:  "quota",
		RoutingCooldownKeyIndex: func() *int {
			v := 0
			return &v
		}(),
	}

	channel.ResetRuntimeState()

	require.Zero(t, channel.TestTime)
	require.Zero(t, channel.ResponseTime)
	require.Zero(t, channel.Balance)
	require.Zero(t, channel.BalanceUpdatedTime)
	require.Zero(t, channel.UsedQuota)
	require.Nil(t, channel.ChannelInfo.MultiKeyStatusList)
	require.Nil(t, channel.ChannelInfo.MultiKeyDisabledReason)
	require.Nil(t, channel.ChannelInfo.MultiKeyDisabledTime)
	require.Zero(t, channel.ChannelInfo.MultiKeyPollingIndex)
	require.False(t, channel.RoutingCooldownActive)
	require.Zero(t, channel.RoutingCooldownResetAt)
	require.Empty(t, channel.RoutingCooldownReason)
	require.Nil(t, channel.RoutingCooldownKeyIndex)

	otherInfo := channel.GetOtherInfo()
	require.Equal(t, map[string]interface{}{
		"source":        "ionet",
		"deployment_id": "dep-1",
	}, otherInfo)
}

func TestResetRuntimeStateClearsOtherInfoWhenOnlyRuntimeKeysRemain(t *testing.T) {
	channel := &Channel{
		OtherInfo: `{"routing_cooldown":{"-1":{"reset_at":123}},"status_reason":"quota","status_time":111}`,
	}

	channel.ResetRuntimeState()

	require.Empty(t, channel.OtherInfo)
	require.Empty(t, channel.GetOtherInfo())
}
