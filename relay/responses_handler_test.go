package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestResponsesCompactRequestPreservesPromptCacheFieldsWhenExpanded(t *testing.T) {
	req := &dto.OpenAIResponsesCompactionRequest{
		Model:                "gpt-5.5",
		Input:                []byte(`"hello"`),
		Instructions:         []byte(`"be helpful"`),
		PreviousResponseID:   "resp_123",
		Store:                []byte(`false`),
		PromptCacheKey:       []byte(`"cache-key-123"`),
		PromptCacheRetention: []byte(`{"scope":"session"}`),
	}

	expanded := &dto.OpenAIResponsesRequest{
		Model:                req.Model,
		Input:                req.Input,
		Instructions:         req.Instructions,
		PreviousResponseID:   req.PreviousResponseID,
		Store:                req.Store,
		PromptCacheKey:       req.PromptCacheKey,
		PromptCacheRetention: req.PromptCacheRetention,
	}

	if string(expanded.PromptCacheKey) != `"cache-key-123"` {
		t.Fatalf("expected prompt_cache_key to be preserved, got %s", string(expanded.PromptCacheKey))
	}
	if string(expanded.PromptCacheRetention) != `{"scope":"session"}` {
		t.Fatalf("expected prompt_cache_retention to be preserved, got %s", string(expanded.PromptCacheRetention))
	}
	if string(expanded.Store) != `false` {
		t.Fatalf("expected store to be preserved, got %s", string(expanded.Store))
	}
}
