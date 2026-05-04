package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildClaudePromptCacheKeyUsesStablePrefix(t *testing.T) {
	reqA := dto.ClaudeRequest{
		Model: "gpt-5.4",
		System: "same system prefix",
		Messages: []dto.ClaudeMessage{
			{Role: "assistant", Content: "previous stable answer"},
			{Role: "user", Content: "tail question A"},
		},
	}
	reqB := dto.ClaudeRequest{
		Model: "gpt-5.4",
		System: "same system prefix",
		Messages: []dto.ClaudeMessage{
			{Role: "assistant", Content: "previous stable answer"},
			{Role: "user", Content: "tail question B"},
		},
	}
	reqC := dto.ClaudeRequest{
		Model: "gpt-5.4",
		System: "changed system prefix",
		Messages: []dto.ClaudeMessage{
			{Role: "assistant", Content: "previous stable answer"},
			{Role: "user", Content: "tail question A"},
		},
	}

	keyA, err := buildClaudePromptCacheKey(nil, reqA)
	require.NoError(t, err)
	keyB, err := buildClaudePromptCacheKey(nil, reqB)
	require.NoError(t, err)
	keyC, err := buildClaudePromptCacheKey(nil, reqC)
	require.NoError(t, err)

	require.NotEmpty(t, keyA)
	require.Equal(t, keyA, keyB)
	require.NotEqual(t, keyA, keyC)
}

func TestClaudeToOpenAIRequestSetsStablePromptCacheKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	req := dto.ClaudeRequest{
		Model: "gpt-5.4",
		System: []dto.ClaudeMediaMessage{
			{
				Type: "text",
				Text: ptrString("same system prefix"),
			},
		},
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "text",
						Text: ptrString("hello"),
					},
				},
			},
		},
	}

	gotA, err := ClaudeToOpenAIRequest(c, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
	require.NoError(t, err)
	gotB, err := ClaudeToOpenAIRequest(c, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
	require.NoError(t, err)

	require.NotEmpty(t, gotA.PromptCacheKey)
	require.Equal(t, gotA.PromptCacheKey, gotB.PromptCacheKey)
}

func TestClaudeToOpenAIRequestPreservesCacheControlForOpenRouterSystemBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	req := dto.ClaudeRequest{
		Model: "claude-3.7-sonnet",
		System: []dto.ClaudeMediaMessage{
			{
				Type:         "text",
				Text:         ptrString("cacheable system block"),
				CacheControl: []byte(`{"type":"ephemeral"}`),
			},
		},
	}

	got, err := ClaudeToOpenAIRequest(c, req, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:  constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-3.7-sonnet",
		},
	})
	require.NoError(t, err)
	require.Len(t, got.Messages, 1)
	parts := got.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	require.Equal(t, `{"type":"ephemeral"}`, string(parts[0].CacheControl))
}

func ptrString(s string) *string {
	return &s
}

func TestClaudeToOpenAIRequestPreservesToolResultImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	imageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
	text := "Read image file 商品图.png"
	req := dto.ClaudeRequest{
		Model: "claude-opus-4-6",
		Messages: []dto.ClaudeMessage{
			{
				Role: "assistant",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "tool_use",
						Id:   "toolu_read_image",
						Name: "Read",
						Input: map[string]any{
							"file_path": "E:\\claudetest\\商品图.png",
						},
					},
				},
			},
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type:      "tool_result",
						ToolUseId: "toolu_read_image",
						Content: []dto.ClaudeMediaMessage{
							{
								Type: "text",
								Text: &text,
							},
							{
								Type: "image",
								Source: &dto.ClaudeMessageSource{
									Type:      "base64",
									MediaType: "image/png",
									Data:      imageData,
								},
							},
						},
					},
				},
			},
		},
	}

	got, err := ClaudeToOpenAIRequest(c, req, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	})
	require.NoError(t, err)
	require.Len(t, got.Messages, 3)

	require.Equal(t, "tool", got.Messages[1].Role)
	require.Equal(t, "toolu_read_image", got.Messages[1].ToolCallId)
	require.Equal(t, text, got.Messages[1].StringContent())

	require.Equal(t, "user", got.Messages[2].Role)
	parts := got.Messages[2].ParseContent()
	require.Len(t, parts, 1)
	require.Equal(t, dto.ContentTypeImageURL, parts[0].Type)
	image := parts[0].GetImageMedia()
	require.NotNil(t, image)
	require.True(t, strings.HasPrefix(image.Url, "data:image/png;base64,"))
	require.Contains(t, image.Url, imageData)
}
