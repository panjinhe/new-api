package service

import "github.com/QuantumNous/new-api/types"

const nginxClientClosedRequestStatusCode = 499

// NormalizeClientCanceledError converts downstream disconnects into a
// non-retryable local error so they do not trigger channel failover,
// cooldown, or noisy error log entries.
func NormalizeClientCanceledError(err *types.NewAPIError) *types.NewAPIError {
	if err == nil {
		return nil
	}
	if !types.IsClientCanceledError(err) {
		return err
	}
	return types.NewError(
		err,
		err.GetErrorCode(),
		types.ErrOptionWithStatusCode(nginxClientClosedRequestStatusCode),
		types.ErrOptionWithSkipRetry(),
		types.ErrOptionWithNoRecordErrorLog(),
	)
}
