package model

const (
	channelOtherInfoStatusReasonKey = "status_reason"
	channelOtherInfoStatusTimeKey   = "status_time"
	channelOtherInfoCodexUsageKey   = "codex_usage_snapshot"
	channelOtherInfoCodexQuotaKey   = "codex_quota_state"
)

var channelRuntimeOtherInfoKeys = map[string]struct{}{
	routingCooldownOtherInfoKey:     {},
	channelOtherInfoCodexUsageKey:   {},
	channelOtherInfoCodexQuotaKey:   {},
	channelOtherInfoStatusReasonKey: {},
	channelOtherInfoStatusTimeKey:   {},
}

// ResetRuntimeState clears per-channel runtime metadata so a newly created
// channel or a channel with replaced credentials does not inherit stale state
// from another account.
func (channel *Channel) ResetRuntimeState() {
	if channel == nil {
		return
	}

	channel.TestTime = 0
	channel.ResponseTime = 0
	channel.Balance = 0
	channel.BalanceUpdatedTime = 0
	channel.UsedQuota = 0

	channel.ChannelInfo.MultiKeyStatusList = nil
	channel.ChannelInfo.MultiKeyDisabledReason = nil
	channel.ChannelInfo.MultiKeyDisabledTime = nil
	channel.ChannelInfo.MultiKeyPollingIndex = 0

	channel.RoutingCooldownActive = false
	channel.RoutingCooldownResetAt = 0
	channel.RoutingCooldownReason = ""
	channel.RoutingCooldownKeyIndex = nil

	otherInfo := channel.GetOtherInfo()
	if len(otherInfo) == 0 {
		channel.OtherInfo = ""
		return
	}
	for key := range channelRuntimeOtherInfoKeys {
		delete(otherInfo, key)
	}
	if len(otherInfo) == 0 {
		channel.OtherInfo = ""
		return
	}
	channel.SetOtherInfo(otherInfo)
}
