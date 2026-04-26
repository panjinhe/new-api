package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func TestShouldRetrySkipsClientCanceledError(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	err := types.NewOpenAIError(
		fmt.Errorf("request context done: %w", context.Canceled),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)
	err = service.NormalizeClientCanceledError(err)

	if shouldRetry(ctx, err, 1) {
		t.Fatalf("expected client canceled error to skip retry")
	}
	if !types.IsSkipRetryError(err) {
		t.Fatalf("expected client canceled error to be marked skip retry")
	}
	if types.IsRecordErrorLog(err) {
		t.Fatalf("expected client canceled error to skip error log recording")
	}
	if err.StatusCode != 499 {
		t.Fatalf("expected status code 499, got %d", err.StatusCode)
	}
}

func TestShouldRetryStillRetriesRegularServerErrors(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	err := types.NewOpenAIError(
		fmt.Errorf("upstream temporarily unavailable"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)

	if !shouldRetry(ctx, err, 1) {
		t.Fatalf("expected normal 500 error to remain retryable")
	}
}

func TestShouldRetrySameChannelAllowsTwoTransientRetries(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI}
	err := types.NewOpenAIError(
		fmt.Errorf("upstream temporarily unavailable"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)

	if !shouldRetrySameChannel(ctx, info, err, 0) {
		t.Fatalf("expected first same-channel retry to be allowed")
	}
	if !shouldRetrySameChannel(ctx, info, err, 1) {
		t.Fatalf("expected second same-channel retry to be allowed")
	}
	if shouldRetrySameChannel(ctx, info, err, 2) {
		t.Fatalf("expected same-channel retry budget to be exhausted")
	}
}

func TestShouldRetrySameChannelIgnoresSpecificChannelCrossRetryBlock(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("specific_channel_id", "9")
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI}
	err := types.NewOpenAIError(
		fmt.Errorf("upstream temporarily unavailable"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)

	if !shouldRetrySameChannel(ctx, info, err, 0) {
		t.Fatalf("expected specific channel requests to allow same-channel retry")
	}
	if shouldRetry(ctx, err, 1) {
		t.Fatalf("expected specific channel requests to block cross-channel retry")
	}
}

func TestShouldRetrySameChannelBlocksNonTransientErrors(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI}

	for _, statusCode := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		err := types.NewOpenAIError(
			fmt.Errorf("status %d", statusCode),
			types.ErrorCodeBadResponseStatusCode,
			statusCode,
		)
		if shouldRetrySameChannel(ctx, info, err, 0) {
			t.Fatalf("expected status %d to skip same-channel retry", statusCode)
		}
	}

	skipErr := types.NewOpenAIError(
		fmt.Errorf("do not retry"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
		types.ErrOptionWithSkipRetry(),
	)
	if shouldRetrySameChannel(ctx, info, skipErr, 0) {
		t.Fatalf("expected skip-retry errors to skip same-channel retry")
	}

	bodyErr := types.NewOpenAIError(
		fmt.Errorf("invalid json body"),
		types.ErrorCodeBadResponseBody,
		http.StatusInternalServerError,
	)
	if shouldRetrySameChannel(ctx, info, bodyErr, 0) {
		t.Fatalf("expected response body parse errors to skip same-channel retry")
	}
}

func TestShouldRetrySameChannelStopsAfterResponseStarted(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	err := types.NewOpenAIError(
		fmt.Errorf("upstream stream interrupted"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Writer.WriteHeaderNow()
	if shouldRetrySameChannel(ctx, &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI}, err, 0) {
		t.Fatalf("expected written responses to skip same-channel retry")
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		RelayFormat:           types.RelayFormatOpenAI,
		ReceivedResponseCount: 1,
	}
	if shouldRetrySameChannel(ctx, info, err, 0) {
		t.Fatalf("expected received stream chunks to skip same-channel retry")
	}
}

func TestShouldRetrySameChannelTaskRelay(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{}

	serverErr := &dto.TaskError{Error: fmt.Errorf("temporary failure"), StatusCode: http.StatusInternalServerError}
	if !shouldRetrySameChannelTaskRelay(ctx, info, serverErr, 0) {
		t.Fatalf("expected task 500 to allow same-channel retry")
	}
	if !shouldRetrySameChannelTaskRelay(ctx, info, serverErr, 1) {
		t.Fatalf("expected task 500 to allow second same-channel retry")
	}
	if shouldRetrySameChannelTaskRelay(ctx, info, serverErr, 2) {
		t.Fatalf("expected task same-channel retry budget to be exhausted")
	}

	rateLimitErr := &dto.TaskError{Error: fmt.Errorf("rate limited"), StatusCode: http.StatusTooManyRequests}
	if !shouldRetrySameChannelTaskRelay(ctx, info, rateLimitErr, 0) {
		t.Fatalf("expected task 429 to allow same-channel retry")
	}

	badRequestErr := &dto.TaskError{Error: fmt.Errorf("bad request"), StatusCode: http.StatusBadRequest}
	if shouldRetrySameChannelTaskRelay(ctx, info, badRequestErr, 0) {
		t.Fatalf("expected task 400 to skip same-channel retry")
	}

	localErr := &dto.TaskError{Error: fmt.Errorf("local error"), StatusCode: http.StatusInternalServerError, LocalError: true}
	if shouldRetrySameChannelTaskRelay(ctx, info, localErr, 0) {
		t.Fatalf("expected local task errors to skip same-channel retry")
	}
}
