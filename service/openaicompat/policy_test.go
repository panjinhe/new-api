package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestShouldChatCompletionsUseResponsesPolicyAlwaysEnabledForCodex(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       false,
		AllChannels:   false,
		ModelPatterns: []string{"^gpt-5\\.4$"},
	}

	if !ShouldChatCompletionsUseResponsesPolicy(policy, 0, constant.ChannelTypeCodex, "gpt-5.4") {
		t.Fatal("expected Codex channel type to always use responses compatibility")
	}
}

func TestShouldChatCompletionsUseResponsesPolicyStillRequiresPolicyForNonCodex(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   false,
		ChannelIDs:    []int{7},
		ModelPatterns: []string{"^gpt-5\\.4$"},
	}

	if ShouldChatCompletionsUseResponsesPolicy(policy, 8, constant.ChannelTypeOpenAI, "gpt-5.4") {
		t.Fatal("expected non-Codex channel outside policy to stay disabled")
	}
	if !ShouldChatCompletionsUseResponsesPolicy(policy, 7, constant.ChannelTypeOpenAI, "gpt-5.4") {
		t.Fatal("expected configured non-Codex channel to remain enabled")
	}
}
