package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetBatchUpdateStoresForTest() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()
	}
}

func TestFlushBatchUpdates_DrainsQuotaAndUsage(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateStoresForTest()
	t.Cleanup(resetBatchUpdateStoresForTest)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() {
		common.BatchUpdateEnabled = false
	})

	user := User{Username: "batch-flush", Quota: 1000}
	require.NoError(t, DB.Create(&user).Error)
	token := Token{Name: "batch-token", Key: "batch-token-key", UserId: user.Id, RemainQuota: 500}
	require.NoError(t, DB.Create(&token).Error)
	channel := Channel{Name: "batch-channel", Type: 1}
	require.NoError(t, DB.Create(&channel).Error)

	require.NoError(t, DecreaseUserQuota(user.Id, 120, false))
	UpdateUserUsedQuotaAndRequestCount(user.Id, 120)
	require.NoError(t, DecreaseTokenQuota(token.Id, token.Key, 120))
	UpdateChannelUsedQuota(channel.Id, 120)

	FlushBatchUpdates()

	var gotUser User
	require.NoError(t, DB.First(&gotUser, user.Id).Error)
	assert.Equal(t, 880, gotUser.Quota)
	assert.Equal(t, 120, gotUser.UsedQuota)
	assert.Equal(t, 1, gotUser.RequestCount)

	var gotToken Token
	require.NoError(t, DB.First(&gotToken, token.Id).Error)
	assert.Equal(t, 380, gotToken.RemainQuota)
	assert.Equal(t, 120, gotToken.UsedQuota)

	var gotChannel Channel
	require.NoError(t, DB.First(&gotChannel, channel.Id).Error)
	assert.Equal(t, int64(120), gotChannel.UsedQuota)

	FlushBatchUpdates()
	require.NoError(t, DB.First(&gotUser, user.Id).Error)
	assert.Equal(t, 880, gotUser.Quota)
	assert.Equal(t, 120, gotUser.UsedQuota)
	assert.Equal(t, 1, gotUser.RequestCount)
}
