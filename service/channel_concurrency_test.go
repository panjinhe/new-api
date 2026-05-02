package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestTryAcquireChannelConcurrencyMemoryLimit(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		channelConcurrencyMu.Lock()
		delete(channelConcurrencyCounts, 1001)
		channelConcurrencyMu.Unlock()
	})

	release, err := TryAcquireChannelConcurrency(1001, 1)
	require.NoError(t, err)
	require.NotNil(t, release)

	_, err = TryAcquireChannelConcurrency(1001, 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrChannelConcurrencySaturated))

	release()

	releaseAgain, err := TryAcquireChannelConcurrency(1001, 1)
	require.NoError(t, err)
	releaseAgain()
}

func TestTryAcquireChannelConcurrencyDisabledWhenLimitIsZero(t *testing.T) {
	release, err := TryAcquireChannelConcurrency(1002, 0)
	require.NoError(t, err)
	require.NotNil(t, release)
	release()
}
