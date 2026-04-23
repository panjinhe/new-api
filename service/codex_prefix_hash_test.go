package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func buildCodexRelayInfoForTest(req dto.Request) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		Request: req,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeCodex,
		},
	}
}

func TestBuildCodexPrefixHashMatchesChatCompatAndResponses(t *testing.T) {
	chatReq := &dto.GeneralOpenAIRequest{
		Model:            "gpt-5.4",
		Messages:         []dto.Message{{Role: "developer", Content: "You are helpful."}, {Role: "user", Content: "hello"}},
		PromptCacheKey:   "pc-key-a",
		ReasoningEffort:  "high",
		Temperature:      nil,
		ParallelTooCalls: nil,
	}
	responsesReq := &dto.OpenAIResponsesRequest{
		Model:          "gpt-5.4",
		Input:          []byte(`[{"role":"user","content":"hello"}]`),
		Instructions:   []byte(`"You are helpful."`),
		PromptCacheKey: []byte(`"pc-key-b"`),
		Reasoning: &dto.Reasoning{
			Effort:  "high",
			Summary: "detailed",
		},
	}

	chatHash, chatBasis, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(chatReq))
	require.NoError(t, err)
	require.Equal(t, "chat_completions_compat", chatBasis)
	require.NotEmpty(t, chatHash)

	responsesHash, responsesBasis, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(responsesReq))
	require.NoError(t, err)
	require.Equal(t, "responses", responsesBasis)
	require.NotEmpty(t, responsesHash)

	require.Equal(t, chatHash, responsesHash)
}

func TestBuildCodexPrefixHashIgnoresPromptCacheKey(t *testing.T) {
	reqA := &dto.OpenAIResponsesRequest{
		Model:          "gpt-5.4",
		Input:          []byte(`[{"role":"user","content":"hello"}]`),
		Instructions:   []byte(`"same instructions"`),
		PromptCacheKey: []byte(`"cache-key-a"`),
	}
	reqB := &dto.OpenAIResponsesRequest{
		Model:          "gpt-5.4",
		Input:          []byte(`[{"role":"user","content":"hello"}]`),
		Instructions:   []byte(`"same instructions"`),
		PromptCacheKey: []byte(`"cache-key-b"`),
	}

	hashA, _, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqA))
	require.NoError(t, err)
	hashB, _, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqB))
	require.NoError(t, err)

	require.Equal(t, hashA, hashB)
}

func TestBuildCodexPrefixHashChangesWhenPrefixChanges(t *testing.T) {
	reqA := &dto.OpenAIResponsesRequest{
		Model:        "gpt-5.4",
		Input:        []byte(`[{"role":"user","content":"hello"}]`),
		Instructions: []byte(`"same instructions"`),
	}
	reqB := &dto.OpenAIResponsesRequest{
		Model:        "gpt-5.4",
		Input:        []byte(`[{"role":"user","content":"hello"}]`),
		Instructions: []byte(`"changed instructions"`),
	}

	hashA, _, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqA))
	require.NoError(t, err)
	hashB, _, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqB))
	require.NoError(t, err)

	require.NotEqual(t, hashA, hashB)
}

func TestBuildCodexPrefixHashIncludesPreviousResponseID(t *testing.T) {
	reqA := &dto.OpenAIResponsesCompactionRequest{
		Model:              "gpt-5.4",
		PreviousResponseID: "resp_1",
	}
	reqB := &dto.OpenAIResponsesCompactionRequest{
		Model:              "gpt-5.4",
		PreviousResponseID: "resp_2",
	}

	hashA, basisA, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqA))
	require.NoError(t, err)
	require.Equal(t, "responses_compact", basisA)
	hashB, basisB, err := buildCodexPrefixHash(buildCodexRelayInfoForTest(reqB))
	require.NoError(t, err)
	require.Equal(t, "responses_compact", basisB)

	require.NotEqual(t, hashA, hashB)
}
