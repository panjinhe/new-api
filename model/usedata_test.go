package model

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertConsumeLogForQuotaDataTest(t *testing.T, log *Log) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(log).Error)
}

func sortQuotaDataForTest(items []*QuotaData) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt < items[j].CreatedAt
		}
		return items[i].ModelName < items[j].ModelName
	})
}

func TestGetQuotaDataByUserId_AggregatesFromConsumeLogs(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&QuotaData{
		UserID:    101,
		Username:  "alice",
		ModelName: "stale-model",
		CreatedAt: 1710000000,
		Count:     99,
		Quota:     99999,
		TokenUsed: 99999,
	}).Error)

	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           101,
		Username:         "alice",
		Type:             LogTypeConsume,
		ModelName:        "gpt-4o",
		CreatedAt:        1710000100,
		PromptTokens:     100,
		CompletionTokens: 50,
		Quota:            200,
	})
	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           101,
		Username:         "alice",
		Type:             LogTypeConsume,
		ModelName:        "gpt-4o",
		CreatedAt:        1710000200,
		PromptTokens:     20,
		CompletionTokens: 30,
		Quota:            50,
	})
	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           101,
		Username:         "alice",
		Type:             LogTypeConsume,
		ModelName:        "gpt-4.1",
		CreatedAt:        1710003700,
		PromptTokens:     10,
		CompletionTokens: 5,
		Quota:            25,
	})
	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           101,
		Username:         "alice",
		Type:             LogTypeRefund,
		ModelName:        "gpt-4o",
		CreatedAt:        1710000400,
		PromptTokens:     999,
		CompletionTokens: 999,
		Quota:            999,
	})

	data, err := GetQuotaDataByUserId(101, 1710000000, 1710007200)
	require.NoError(t, err)

	sortQuotaDataForTest(data)
	require.Len(t, data, 2)

	assert.Equal(t, "gpt-4o", data[0].ModelName)
	assert.Equal(t, int64(1710000000), data[0].CreatedAt)
	assert.Equal(t, 2, data[0].Count)
	assert.Equal(t, 250, data[0].Quota)
	assert.Equal(t, 200, data[0].TokenUsed)

	assert.Equal(t, "gpt-4.1", data[1].ModelName)
	assert.Equal(t, int64(1710003600), data[1].CreatedAt)
	assert.Equal(t, 1, data[1].Count)
	assert.Equal(t, 25, data[1].Quota)
	assert.Equal(t, 15, data[1].TokenUsed)
}

func TestGetQuotaDataByUsername_AggregatesFromConsumeLogs(t *testing.T) {
	truncateTables(t)

	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           201,
		Username:         "bob",
		Type:             LogTypeConsume,
		ModelName:        "claude-3-7-sonnet",
		CreatedAt:        1720000001,
		PromptTokens:     11,
		CompletionTokens: 22,
		Quota:            33,
	})
	insertConsumeLogForQuotaDataTest(t, &Log{
		UserId:           202,
		Username:         "other-user",
		Type:             LogTypeConsume,
		ModelName:        "claude-3-7-sonnet",
		CreatedAt:        1720000002,
		PromptTokens:     99,
		CompletionTokens: 99,
		Quota:            99,
	})

	data, err := GetQuotaDataByUsername("bob", 1719999000, 1720003600)
	require.NoError(t, err)
	require.Len(t, data, 1)

	assert.Equal(t, "claude-3-7-sonnet", data[0].ModelName)
	assert.Equal(t, int64(1719997200), data[0].CreatedAt)
	assert.Equal(t, 1, data[0].Count)
	assert.Equal(t, 33, data[0].Quota)
	assert.Equal(t, 33, data[0].TokenUsed)
}
