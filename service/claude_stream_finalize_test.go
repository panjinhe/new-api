package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestStreamResponseOpenAI2Claude_FinalizesStopChunkWithCachedUsage(t *testing.T) {
	finishReason := "stop"
	info := &relaycommon.RelayInfo{
		SendResponseCount: 2,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeText,
			Index:            0,
			Usage: &dto.Usage{
				PromptTokens:     6,
				CompletionTokens: 2,
				TotalTokens:      8,
			},
		},
	}

	streamResp := &dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl-local",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-5.4",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				FinishReason: common.GetPointer(finishReason),
			},
		},
	}

	claudeResponses := StreamResponseOpenAI2Claude(streamResp, info)
	if len(claudeResponses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(claudeResponses))
	}
	if claudeResponses[0].Type != "content_block_stop" {
		t.Fatalf("expected first response to stop content block, got %q", claudeResponses[0].Type)
	}
	if claudeResponses[1].Type != "message_delta" {
		t.Fatalf("expected second response to be message_delta, got %q", claudeResponses[1].Type)
	}
	if claudeResponses[1].Usage == nil || claudeResponses[1].Usage.OutputTokens != 2 {
		t.Fatal("expected message_delta to carry cached usage")
	}
	if claudeResponses[2].Type != "message_stop" {
		t.Fatalf("expected final response to be message_stop, got %q", claudeResponses[2].Type)
	}
}
