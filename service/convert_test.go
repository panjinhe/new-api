package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
