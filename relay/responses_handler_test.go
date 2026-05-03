package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
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

func TestShouldUseRawOpenAIResponsesBodyOnlyForOpenAICompatiblePlainResponses(t *testing.T) {
	base := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType:     appconstant.APITypeOpenAI,
			ChannelType: appconstant.ChannelTypeOpenAI,
		},
	}
	if !shouldUseRawOpenAIResponsesBody(base) {
		t.Fatalf("expected raw responses body fast path for OpenAI responses")
	}

	mapped := *base
	mapped.IsModelMapped = true
	if shouldUseRawOpenAIResponsesBody(&mapped) {
		t.Fatalf("expected model mapped request to use converted path")
	}

	compact := *base
	compact.RelayMode = relayconstant.RelayModeResponsesCompact
	if shouldUseRawOpenAIResponsesBody(&compact) {
		t.Fatalf("expected compact responses request to use converted path")
	}

	gemini := *base
	gemini.ChannelMeta = &relaycommon.ChannelMeta{
		ApiType:     appconstant.APITypeGemini,
		ChannelType: appconstant.ChannelTypeGemini,
	}
	if shouldUseRawOpenAIResponsesBody(&gemini) {
		t.Fatalf("expected non OpenAI-compatible channel to use converted path")
	}
}

func TestBuildRawOpenAIResponsesBodyPatchesReasoningSuffixAndPreservesCacheKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gpt-5.5-xhigh","input":"hello","prompt_cache_key":"cache-key-123"}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req := &dto.OpenAIResponsesRequest{Model: "gpt-5.5-xhigh"}
	info := &relaycommon.RelayInfo{
		Request: req,
	}

	out, err := buildRawOpenAIResponsesBody(ctx, info)
	if err != nil {
		t.Fatalf("buildRawOpenAIResponsesBody returned error: %v", err)
	}
	if got := gjson.GetBytes(out, "model").String(); got != "gpt-5.5" {
		t.Fatalf("expected model suffix to be stripped, got %q", got)
	}
	if got := gjson.GetBytes(out, "reasoning.effort").String(); got != "xhigh" {
		t.Fatalf("expected reasoning effort xhigh, got %q", got)
	}
	if got := gjson.GetBytes(out, "prompt_cache_key").String(); got != "cache-key-123" {
		t.Fatalf("expected prompt_cache_key to be preserved, got %q", got)
	}
	if info.ReasoningEffort != "xhigh" {
		t.Fatalf("expected info reasoning effort xhigh, got %q", info.ReasoningEffort)
	}
}

func TestRawOpenAIResponsesBodyParamOverrideSyncsPromptCacheKeyToSessionID(t *testing.T) {
	input := []byte(`{"model":"gpt-5.5","input":"hello","prompt_cache_key":"cache-body"}`)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "sync_fields",
						"from": "header:session_id",
						"to":   "json:prompt_cache_key",
					},
				},
			},
		},
	}

	out, err := relaycommon.ApplyParamOverrideWithRelayInfo(input, info)
	if err != nil {
		t.Fatalf("ApplyParamOverrideWithRelayInfo returned error: %v", err)
	}
	if got := gjson.GetBytes(out, "prompt_cache_key").String(); got != "cache-body" {
		t.Fatalf("expected prompt_cache_key to remain stable, got %q", got)
	}
	if !info.UseRuntimeHeadersOverride {
		t.Fatalf("expected runtime headers to be enabled")
	}
	if got := common.Interface2String(info.RuntimeHeadersOverride["session_id"]); got != "cache-body" {
		t.Fatalf("expected session_id to sync from prompt_cache_key, got %q", got)
	}
}
