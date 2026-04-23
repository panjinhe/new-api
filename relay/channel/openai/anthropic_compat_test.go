package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiStreamHandler_ClaudeFormatFinalizesOnEOFWithoutStopChunk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx.Set(common.RequestIdKey, "claude-eof")

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	streamBody := strings.Join([]string{
		`data: {"id":"chatcmpl-eof","object":"chat.completion.chunk","created":1710000000,"model":"gpt-5.4","choices":[{"index":0,"delta":{"role":"assistant","content":""}}]}`,
		``,
		`data: {"id":"chatcmpl-eof","object":"chat.completion.chunk","created":1710000000,"model":"gpt-5.4","choices":[{"index":0,"delta":{"content":"OK"}}]}`,
		``,
	}, "\n")
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(streamBody)),
	}

	usage, err := OaiStreamHandler(ctx, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	body := recorder.Body.String()
	require.Contains(t, body, "event: content_block_start")
	require.Contains(t, body, "event: content_block_delta")
	require.Contains(t, body, "event: content_block_stop")
	require.Contains(t, body, "event: message_delta")
	require.Contains(t, body, "event: message_stop")

	contentStartIdx := strings.Index(body, "event: content_block_start")
	contentStopIdx := strings.Index(body, "event: content_block_stop")
	messageStopIdx := strings.Index(body, "event: message_stop")
	require.NotEqual(t, -1, contentStartIdx)
	require.NotEqual(t, -1, contentStopIdx)
	require.NotEqual(t, -1, messageStopIdx)
	require.Less(t, contentStartIdx, contentStopIdx)
	require.Less(t, contentStopIdx, messageStopIdx)
}

func TestOaiResponsesToChatHandler_AggregatesEventStreamBodyForClaude(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx.Set(common.RequestIdKey, "responses-sse")

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	streamBody := strings.Join([]string{
		`event: response.created`,
		`data: {"type":"response.created","response":{"id":"resp_1","object":"response","created_at":1710000000,"model":"gpt-5.4","output":[]}}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"OK"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","created_at":1710000000,"status":"completed","model":"gpt-5.4","output":[{"type":"message","id":"msg_1","status":"completed","role":"assistant","content":[{"type":"output_text","text":"OK","annotations":[]}]}],"usage":{"input_tokens":12,"output_tokens":4,"total_tokens":16}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(streamBody)),
	}

	usage, err := OaiResponsesToChatHandler(ctx, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 12, usage.PromptTokens)
	require.Equal(t, 4, usage.CompletionTokens)
	require.Equal(t, 16, usage.TotalTokens)

	var claudeResp dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &claudeResp))
	require.Equal(t, "message", claudeResp.Type)
	require.Equal(t, "assistant", claudeResp.Role)
	require.Equal(t, "gpt-5.4", claudeResp.Model)
	require.Equal(t, "end_turn", claudeResp.StopReason)
	require.Len(t, claudeResp.Content, 1)
	require.Equal(t, "text", claudeResp.Content[0].Type)
	require.Equal(t, "OK", claudeResp.Content[0].GetText())
	require.NotNil(t, claudeResp.Usage)
	require.Equal(t, 12, claudeResp.Usage.InputTokens)
	require.Equal(t, 4, claudeResp.Usage.OutputTokens)
}
