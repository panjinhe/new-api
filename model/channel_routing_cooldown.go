package model

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
)

const (
	routingCooldownOtherInfoKey = "routing_cooldown"
	routingCooldownSingleKeyIdx = -1
)

type RoutingCooldownState struct {
	Kind      string `json:"kind,omitempty"`
	Reason    string `json:"reason,omitempty"`
	ResetAt   int64  `json:"reset_at,omitempty"`
	Source    string `json:"source,omitempty"`
	KeyIndex  int    `json:"key_index"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

func normalizeRoutingCooldownKeyIndex(isMultiKey bool, keyIndex int) int {
	if !isMultiKey || keyIndex < 0 {
		return routingCooldownSingleKeyIdx
	}
	return keyIndex
}

func routingCooldownEntryKey(keyIndex int) string {
	return strconv.Itoa(keyIndex)
}

func (channel *Channel) GetRoutingCooldownStates() map[string]RoutingCooldownState {
	if channel == nil {
		return map[string]RoutingCooldownState{}
	}
	otherInfo := channel.GetOtherInfo()
	raw, ok := otherInfo[routingCooldownOtherInfoKey]
	if !ok || raw == nil {
		return map[string]RoutingCooldownState{}
	}

	encoded, err := common.Marshal(raw)
	if err != nil {
		return map[string]RoutingCooldownState{}
	}

	var states map[string]RoutingCooldownState
	if err := common.Unmarshal(encoded, &states); err != nil || states == nil {
		return map[string]RoutingCooldownState{}
	}
	return states
}

func (channel *Channel) SetRoutingCooldownStates(states map[string]RoutingCooldownState) {
	if channel == nil {
		return
	}
	otherInfo := channel.GetOtherInfo()
	if len(states) == 0 {
		delete(otherInfo, routingCooldownOtherInfoKey)
	} else {
		otherInfo[routingCooldownOtherInfoKey] = states
	}
	channel.SetOtherInfo(otherInfo)
}

func isRoutingCooldownActive(state RoutingCooldownState, now int64) bool {
	if state.ResetAt == 0 {
		return false
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	return state.ResetAt > now
}

func (channel *Channel) HasRoutingCooldown(now int64) bool {
	for _, state := range channel.GetRoutingCooldownStates() {
		if isRoutingCooldownActive(state, now) {
			return true
		}
	}
	return false
}

func (channel *Channel) IsKeyRoutingCooledDown(keyIndex int, now int64) bool {
	if channel == nil {
		return false
	}
	keyIndex = normalizeRoutingCooldownKeyIndex(channel.ChannelInfo.IsMultiKey, keyIndex)
	states := channel.GetRoutingCooldownStates()
	if state, ok := states[routingCooldownEntryKey(routingCooldownSingleKeyIdx)]; ok && isRoutingCooldownActive(state, now) {
		return true
	}
	if keyIndex == routingCooldownSingleKeyIdx {
		return false
	}
	if state, ok := states[routingCooldownEntryKey(keyIndex)]; ok && isRoutingCooldownActive(state, now) {
		return true
	}
	return false
}

func (channel *Channel) HasAvailableRoute(now int64) bool {
	if channel == nil || channel.Status != common.ChannelStatusEnabled {
		return false
	}
	if !channel.ChannelInfo.IsMultiKey {
		return channel.Key != "" && !channel.IsKeyRoutingCooledDown(routingCooldownSingleKeyIdx, now)
	}

	keys := channel.GetKeys()
	if len(keys) == 0 {
		return false
	}

	statusList := channel.ChannelInfo.MultiKeyStatusList
	for idx := range keys {
		status := common.ChannelStatusEnabled
		if statusList != nil {
			if s, ok := statusList[idx]; ok {
				status = s
			}
		}
		if status != common.ChannelStatusEnabled {
			continue
		}
		if channel.IsKeyRoutingCooledDown(idx, now) {
			continue
		}
		return true
	}
	return false
}

func (channel *Channel) RoutingCooldownSnapshot(now int64) (RoutingCooldownState, bool) {
	states := channel.GetRoutingCooldownStates()
	var picked RoutingCooldownState
	found := false
	for _, state := range states {
		if !isRoutingCooldownActive(state, now) {
			continue
		}
		if !found || state.ResetAt < picked.ResetAt {
			picked = state
			found = true
		}
	}
	return picked, found
}

func (channel *Channel) ApplyRoutingCooldownView(now int64) {
	if channel == nil {
		return
	}
	channel.RoutingCooldownActive = false
	channel.RoutingCooldownResetAt = 0
	channel.RoutingCooldownReason = ""
	channel.RoutingCooldownKeyIndex = nil

	state, ok := channel.RoutingCooldownSnapshot(now)
	if !ok {
		return
	}
	channel.RoutingCooldownActive = true
	channel.RoutingCooldownResetAt = state.ResetAt
	channel.RoutingCooldownReason = state.Reason
	if state.KeyIndex >= 0 {
		keyIndex := state.KeyIndex
		channel.RoutingCooldownKeyIndex = &keyIndex
	}
}

func (channel *Channel) DropExpiredRoutingCooldowns(now int64) bool {
	if channel == nil {
		return false
	}
	states := channel.GetRoutingCooldownStates()
	if len(states) == 0 {
		return false
	}
	changed := false
	for key, state := range states {
		if isRoutingCooldownActive(state, now) {
			continue
		}
		delete(states, key)
		changed = true
	}
	if changed {
		channel.SetRoutingCooldownStates(states)
	}
	return changed
}
