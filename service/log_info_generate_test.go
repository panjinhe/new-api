package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfoIncludesTimingBreakdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	start := time.Unix(100, 0)
	info := &relaycommon.RelayInfo{
		StartTime:                  start,
		FirstResponseTime:          start.Add(2500 * time.Millisecond),
		UpstreamRequestStartTime:   start.Add(100 * time.Millisecond),
		UpstreamRequestWroteTime:   start.Add(250 * time.Millisecond),
		UpstreamResponseHeaderTime: start.Add(900 * time.Millisecond),
		UpstreamFirstByteTime:      start.Add(880 * time.Millisecond),
		UpstreamGotConnTime:        start.Add(120 * time.Millisecond),
		UpstreamReusedConn:         true,
		ChannelMeta:                &relaycommon.ChannelMeta{},
		Request:                    &dto.GeneralOpenAIRequest{},
	}

	other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, -1, -1)
	timing, ok := other["timing"].(map[string]interface{})
	require.True(t, ok)
	require.EqualValues(t, 100, timing["gateway_to_upstream_ms"])
	require.EqualValues(t, 150, timing["upstream_request_write_ms"])
	require.EqualValues(t, 800, timing["upstream_header_ms"])
	require.EqualValues(t, 780, timing["upstream_first_byte_ms"])
	require.EqualValues(t, 1600, timing["upstream_body_first_ms"])
	require.EqualValues(t, 2500, timing["total_first_response_ms"])
	require.EqualValues(t, 20, timing["got_conn_ms"])
	require.Equal(t, true, timing["upstream_reused_conn"])
}
