package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerMergesRelevantHeadersIntoMetadata(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"error": {
				"message": "usage limit reached",
				"type": "rate_limit_error",
				"code": "rate_limit_exceeded"
			}
		}`)),
	}
	resp.Header.Set("Retry-After", "60")

	apiErr := RelayErrorHandler(context.Background(), resp, false)
	require.NotNil(t, apiErr)

	metadata := make(map[string]interface{})
	require.NoError(t, common.Unmarshal(apiErr.Metadata, &metadata))
	assert.Equal(t, "60", metadata["retry_after"])
	assert.EqualValues(t, 60, metadata["retry_after_seconds"])
	assert.NotZero(t, metadata["reset_at"])
}
