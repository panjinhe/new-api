package codex

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestConvertOpenAIResponsesRequestSyncsPromptCacheKeyToSessionID(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	promptCacheKey, err := common.Marshal("cache-session-123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:          "gpt-5.4",
		PromptCacheKey: promptCacheKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !info.UseRuntimeHeadersOverride {
		t.Fatal("expected runtime header override to be enabled")
	}
	if got := common.Interface2String(info.RuntimeHeadersOverride["session_id"]); got != "cache-session-123" {
		t.Fatalf("expected session_id from prompt_cache_key, got %q", got)
	}
}

func TestConvertOpenAIResponsesRequestKeepsExistingSessionID(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta:               &relaycommon.ChannelMeta{},
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]interface{}{
			"session_id": "client-session",
		},
	}
	promptCacheKey, err := common.Marshal("cache-session-123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:          "gpt-5.4",
		PromptCacheKey: promptCacheKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := common.Interface2String(info.RuntimeHeadersOverride["session_id"]); got != "client-session" {
		t.Fatalf("expected existing session_id to be preserved, got %q", got)
	}
}
