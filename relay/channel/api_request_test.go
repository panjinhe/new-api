package channel

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	common2 "github.com/QuantumNous/new-api/common"
	appconstant "github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}

func TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	_, ok := headers["x-upstream-trace"]
	require.False(t, ok)
}

func TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-upstream-trace"])
}

func TestProcessHeaderOverride_RuntimeOverrideIsFinalHeaderMap(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsChannelTest:             false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-static":  "runtime-value",
			"x-runtime": "runtime-only",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
				"X-Legacy": "legacy-only",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "runtime-value", headers["x-static"])
	require.Equal(t, "runtime-only", headers["x-runtime"])
	_, exists := headers["x-legacy"]
	require.False(t, exists)
}

func TestProcessHeaderOverride_PassthroughSkipsAcceptEncoding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-trace-id"])

	_, hasAcceptEncoding := headers["accept-encoding"]
	require.False(t, hasAcceptEncoding)
}

func TestProcessHeaderOverride_PassHeadersTemplateSetsRuntimeHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Originator", "Codex CLI")
	ctx.Request.Header.Set("Session_id", "sess-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		RequestHeaders: map[string]string{
			"Originator": "Codex CLI",
			"Session_id": "sess-123",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]any{
				"operations": []any{
					map[string]any{
						"mode":  "pass_headers",
						"value": []any{"Originator", "Session_id", "X-Codex-Beta-Features"},
					},
				},
			},
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"gpt-4.1"}`), info)
	require.NoError(t, err)
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "Codex CLI", info.RuntimeHeadersOverride["originator"])
	require.Equal(t, "sess-123", info.RuntimeHeadersOverride["session_id"])
	_, exists := info.RuntimeHeadersOverride["x-codex-beta-features"]
	require.False(t, exists)
	require.Equal(t, "legacy-value", info.RuntimeHeadersOverride["x-static"])

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "Codex CLI", headers["originator"])
	require.Equal(t, "sess-123", headers["session_id"])
	_, exists = headers["x-codex-beta-features"]
	require.False(t, exists)

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	applyHeaderOverrideToRequest(upstreamReq, headers)
	require.Equal(t, "Codex CLI", upstreamReq.Header.Get("Originator"))
	require.Equal(t, "sess-123", upstreamReq.Header.Get("Session_id"))
	require.Empty(t, upstreamReq.Header.Get("X-Codex-Beta-Features"))
}

func TestMaybeCompressUpstreamRequestBodyCompressesAllowedLargeJSON(t *testing.T) {
	restoreUpstreamGzipTestConfig(t, true, "54", 128, gzip.BestSpeed)

	body := []byte(`{"model":"gpt-5.4","input":"` + strings.Repeat("hello world ", 4096) + `"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 54,
			ApiType:   appconstant.APITypeOpenAI,
		},
	}

	require.NoError(t, maybeCompressUpstreamRequestBody(req, info))
	require.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
	require.True(t, info.UpstreamRequestGzipEnabled)
	require.Equal(t, len(body), info.UpstreamRequestGzipBytesIn)
	require.Less(t, info.UpstreamRequestGzipBytesOut, len(body))
	require.Greater(t, info.UpstreamRequestGzipRatio, 0.0)
	require.Less(t, info.UpstreamRequestGzipRatio, 1.0)

	gr, err := gzip.NewReader(req.Body)
	require.NoError(t, err)
	defer gr.Close()
	decompressed, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.Equal(t, body, decompressed)
}

func TestMaybeCompressUpstreamRequestBodySkipsWhenDisabled(t *testing.T) {
	restoreUpstreamGzipTestConfig(t, false, "54", 128, gzip.BestSpeed)

	body := []byte(`{"input":"` + strings.Repeat("hello world ", 1024) + `"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 54,
			ApiType:   appconstant.APITypeOpenAI,
		},
	}

	require.NoError(t, maybeCompressUpstreamRequestBody(req, info))
	require.Empty(t, req.Header.Get("Content-Encoding"))
	require.False(t, info.UpstreamRequestGzipEnabled)
	require.Equal(t, int64(len(body)), req.ContentLength)
}

func TestMaybeCompressUpstreamRequestBodySkipsUnlistedChannel(t *testing.T) {
	restoreUpstreamGzipTestConfig(t, true, "55", 128, gzip.BestSpeed)

	body := []byte(`{"input":"` + strings.Repeat("hello world ", 1024) + `"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 54,
			ApiType:   appconstant.APITypeOpenAI,
		},
	}

	require.NoError(t, maybeCompressUpstreamRequestBody(req, info))
	require.Empty(t, req.Header.Get("Content-Encoding"))
	require.False(t, info.UpstreamRequestGzipEnabled)
}

func TestMaybeCompressUpstreamRequestBodySkipsSmallBody(t *testing.T) {
	restoreUpstreamGzipTestConfig(t, true, "54", 1<<20, gzip.BestSpeed)

	body := []byte(`{"input":"small"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 54,
			ApiType:   appconstant.APITypeOpenAI,
		},
	}

	require.NoError(t, maybeCompressUpstreamRequestBody(req, info))
	require.Empty(t, req.Header.Get("Content-Encoding"))
	require.False(t, info.UpstreamRequestGzipEnabled)
}

func restoreUpstreamGzipTestConfig(t *testing.T, enabled bool, channelIDs string, minBytes int, level int) {
	t.Helper()
	oldEnabled := common2.UpstreamRequestGzipEnabled
	oldChannelIDs := common2.UpstreamRequestGzipChannelIDs
	oldMinBytes := common2.UpstreamRequestGzipMinBytes
	oldLevel := common2.UpstreamRequestGzipLevel
	common2.UpstreamRequestGzipEnabled = enabled
	common2.UpstreamRequestGzipChannelIDs = channelIDs
	common2.UpstreamRequestGzipMinBytes = minBytes
	common2.UpstreamRequestGzipLevel = level
	t.Cleanup(func() {
		common2.UpstreamRequestGzipEnabled = oldEnabled
		common2.UpstreamRequestGzipChannelIDs = oldChannelIDs
		common2.UpstreamRequestGzipMinBytes = oldMinBytes
		common2.UpstreamRequestGzipLevel = oldLevel
	})
}
