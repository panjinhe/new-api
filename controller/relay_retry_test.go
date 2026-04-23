package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
