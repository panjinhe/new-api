package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneralErrorResponseTryToOpenAIErrorPreservesTopLevelMetadata(t *testing.T) {
	resp := GeneralErrorResponse{
		Error:    []byte(`{"message":"usage limit reached","type":"rate_limit_error","code":"rate_limit_exceeded"}`),
		Metadata: []byte(`{"reset_at":2000000000,"limit_window_seconds":18000}`),
	}

	openAIError := resp.TryToOpenAIError()
	require.NotNil(t, openAIError)
	require.JSONEq(t, `{"reset_at":2000000000,"limit_window_seconds":18000}`, string(openAIError.Metadata))
}
