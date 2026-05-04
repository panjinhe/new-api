package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestTryAcquireModelRequestConcurrencyMemoryLimit(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		modelRequestConcurrencyMu.Lock()
		delete(modelRequestConcurrencyCounts, 2001)
		modelRequestConcurrencyMu.Unlock()
	})

	release, err := TryAcquireModelRequestConcurrency(2001, 1)
	require.NoError(t, err)
	require.NotNil(t, release)

	_, err = TryAcquireModelRequestConcurrency(2001, 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrModelRequestConcurrencySaturated))

	release()

	releaseAgain, err := TryAcquireModelRequestConcurrency(2001, 1)
	require.NoError(t, err)
	releaseAgain()
}
