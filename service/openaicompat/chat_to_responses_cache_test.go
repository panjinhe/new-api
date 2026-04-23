package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestChatCompletionsRequestToResponsesRequestPreservesPromptCacheFields(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model:                "gpt-5.5",
		Messages:             []dto.Message{{Role: "user", Content: "hello"}},
		PromptCacheKey:       "cache-key-123",
		PromptCacheRetention: []byte(`{"scope":"session"}`),
	}

	respReq, err := ChatCompletionsRequestToResponsesRequest(req)
	if err != nil {
		t.Fatalf("expected conversion to succeed, got error: %v", err)
	}

	var promptCacheKey string
	if err := common.Unmarshal(respReq.PromptCacheKey, &promptCacheKey); err != nil {
		t.Fatalf("expected prompt_cache_key to be preserved, got unmarshal error: %v", err)
	}
	if promptCacheKey != "cache-key-123" {
		t.Fatalf("expected prompt_cache_key to be preserved, got %q", promptCacheKey)
	}
	if string(respReq.PromptCacheRetention) != `{"scope":"session"}` {
		t.Fatalf("expected prompt_cache_retention to be preserved, got %s", string(respReq.PromptCacheRetention))
	}
}
