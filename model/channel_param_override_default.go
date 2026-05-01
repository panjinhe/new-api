package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

var defaultOpenAIParamOverride = map[string]interface{}{
	"operations": []map[string]interface{}{
		{
			"mode": "pass_headers",
			"value": []string{
				"Originator",
				"Session_id",
				"User-Agent",
				"X-Codex-Beta-Features",
				"X-Codex-Turn-Metadata",
			},
			"keep_origin": true,
		},
		{
			"mode": "sync_fields",
			"from": "header:session_id",
			"to":   "json:prompt_cache_key",
		},
	},
}

// ApplyDefaultParamOverrideForNewChannel gives new OpenAI-compatible proxy
// channels stable cache/session forwarding unless the admin supplied an override.
func ApplyDefaultParamOverrideForNewChannel(channel *Channel) bool {
	if channel == nil {
		return false
	}
	if channel.Type != constant.ChannelTypeOpenAI {
		return false
	}
	if channel.ParamOverride != nil && strings.TrimSpace(*channel.ParamOverride) != "" {
		return false
	}

	data, err := common.Marshal(defaultOpenAIParamOverride)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to marshal default OpenAI param override: %v", err))
		return false
	}
	channel.ParamOverride = common.GetPointer(string(data))
	return true
}
