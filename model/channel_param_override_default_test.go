package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultParamOverrideForNewOpenAIChannel(t *testing.T) {
	channel := &Channel{
		Type:   constant.ChannelTypeOpenAI,
		Name:   "openai-compatible",
		Key:    "sk-test",
		Status: 1,
		Models: "gpt-5.4",
		Group:  "default",
	}

	changed := ApplyDefaultParamOverrideForNewChannel(channel)
	require.True(t, changed)
	require.NotNil(t, channel.ParamOverride)

	var parsed map[string]interface{}
	require.NoError(t, common.Unmarshal([]byte(*channel.ParamOverride), &parsed))
	require.Contains(t, parsed, "operations")
}

func TestApplyDefaultParamOverridePreservesExplicitOverride(t *testing.T) {
	explicit := `{"temperature":0.2}`
	channel := &Channel{
		Type:          constant.ChannelTypeOpenAI,
		ParamOverride: &explicit,
	}

	changed := ApplyDefaultParamOverrideForNewChannel(channel)
	require.False(t, changed)
	require.Equal(t, explicit, *channel.ParamOverride)
}

func TestApplyDefaultParamOverrideSkipsCodexChannel(t *testing.T) {
	channel := &Channel{
		Type: constant.ChannelTypeCodex,
	}

	changed := ApplyDefaultParamOverrideForNewChannel(channel)
	require.False(t, changed)
	require.Nil(t, channel.ParamOverride)
}

func TestBatchInsertChannelsAppliesDefaultParamOverride(t *testing.T) {
	truncateTables(t)
	setProxyEnv(t, "")

	inserted := Channel{
		Type:   constant.ChannelTypeOpenAI,
		Name:   "inserted-openai-compatible",
		Key:    "sk-test",
		Status: 1,
		Models: "gpt-5.4",
		Group:  "default",
	}
	require.NoError(t, BatchInsertChannels([]Channel{inserted}))

	var loaded Channel
	require.NoError(t, DB.Where("name = ?", "inserted-openai-compatible").First(&loaded).Error)
	require.NotNil(t, loaded.ParamOverride)
	require.Contains(t, *loaded.ParamOverride, "prompt_cache_key")
	require.Contains(t, *loaded.ParamOverride, "session_id")
}
