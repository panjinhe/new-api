package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	openaicompat "github.com/QuantumNous/new-api/service/openaicompat"
)

func appendCodexPrefixHashAdminInfo(relayInfo *relaycommon.RelayInfo, adminInfo map[string]interface{}) {
	if adminInfo == nil {
		return
	}

	hash, basis, err := buildCodexPrefixHash(relayInfo)
	if err != nil {
		adminInfo["codex_prefix_hash_error"] = err.Error()
		if basis != "" {
			adminInfo["codex_prefix_hash_basis"] = basis
		}
		return
	}
	if hash == "" {
		return
	}

	adminInfo["codex_prefix_hash"] = hash
	if basis != "" {
		adminInfo["codex_prefix_hash_basis"] = basis
	}
}

func buildCodexPrefixHash(relayInfo *relaycommon.RelayInfo) (string, string, error) {
	if !shouldTrackCodexPrefixHash(relayInfo) {
		return "", "", nil
	}

	normalized, basis, err := normalizeCodexPrefixPayload(relayInfo.Request)
	if err != nil {
		return "", basis, err
	}
	if len(normalized) == 0 {
		return "", basis, nil
	}

	data, err := common.Marshal(normalized)
	if err != nil {
		return "", basis, fmt.Errorf("marshal normalized codex prefix payload: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16]), basis, nil
}

func shouldTrackCodexPrefixHash(relayInfo *relaycommon.RelayInfo) bool {
	if relayInfo == nil || relayInfo.Request == nil || relayInfo.ChannelMeta == nil {
		return false
	}
	return relayInfo.ChannelType == constant.ChannelTypeCodex
}

func normalizeCodexPrefixPayload(request dto.Request) (map[string]interface{}, string, error) {
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		responsesReq, err := openaicompat.ChatCompletionsRequestToResponsesRequest(req)
		if err != nil {
			return nil, "chat_completions_compat", fmt.Errorf("normalize chat completions request to responses request: %w", err)
		}
		return normalizeResponsesPrefixPayload(responsesReq), "chat_completions_compat", nil
	case *dto.OpenAIResponsesRequest:
		return normalizeResponsesPrefixPayload(req), "responses", nil
	case *dto.OpenAIResponsesCompactionRequest:
		return normalizeResponsesPrefixPayload(&dto.OpenAIResponsesRequest{
			Model:                req.Model,
			Input:                req.Input,
			Instructions:         req.Instructions,
			PreviousResponseID:   req.PreviousResponseID,
			Store:                req.Store,
			PromptCacheKey:       req.PromptCacheKey,
			PromptCacheRetention: req.PromptCacheRetention,
		}), "responses_compact", nil
	default:
		return nil, "", nil
	}
}

func normalizeResponsesPrefixPayload(req *dto.OpenAIResponsesRequest) map[string]interface{} {
	if req == nil {
		return nil
	}

	payload := map[string]interface{}{}
	if model := strings.TrimSpace(req.Model); model != "" {
		payload["model"] = model
	}
	appendNormalizedRawMessage(payload, "conversation", req.Conversation)
	appendNormalizedRawMessage(payload, "context_management", req.ContextManagement)
	appendNormalizedRawMessage(payload, "input", req.Input)
	appendNormalizedRawMessage(payload, "instructions", req.Instructions)
	appendNormalizedRawMessage(payload, "prompt", req.Prompt)
	appendNormalizedRawMessage(payload, "text", req.Text)
	appendNormalizedRawMessage(payload, "tool_choice", req.ToolChoice)
	appendNormalizedRawMessage(payload, "tools", req.Tools)
	appendNormalizedRawMessage(payload, "truncation", req.Truncation)
	if reasoning := normalizeReasoning(req.Reasoning); len(reasoning) > 0 {
		payload["reasoning"] = reasoning
	}
	if prev := strings.TrimSpace(req.PreviousResponseID); prev != "" {
		payload["previous_response_id"] = prev
	}
	return payload
}

func normalizeReasoning(reasoning *dto.Reasoning) map[string]interface{} {
	if reasoning == nil {
		return nil
	}
	out := map[string]interface{}{}
	if effort := strings.TrimSpace(reasoning.Effort); effort != "" {
		out["effort"] = effort
	}
	if summary := strings.TrimSpace(reasoning.Summary); summary != "" {
		out["summary"] = summary
	}
	return out
}

func appendNormalizedRawMessage(target map[string]interface{}, key string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	normalized, ok := normalizeRawMessage(raw)
	if !ok {
		return
	}
	target[key] = normalized
}

func normalizeRawMessage(raw json.RawMessage) (interface{}, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, false
	}

	var value interface{}
	if err := common.Unmarshal(raw, &value); err != nil {
		return trimmed, true
	}
	return value, true
}
