package model

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func setProxyEnv(t *testing.T, value string) {
	t.Helper()
	for _, key := range []string{"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "https_proxy", "http_proxy", "all_proxy"} {
		prev, ok := os.LookupEnv(key)
		if value == "" {
			require.NoError(t, os.Unsetenv(key))
		} else {
			require.NoError(t, os.Setenv(key, value))
		}
		t.Cleanup(func() {
			if !ok {
				_ = os.Unsetenv(key)
				return
			}
			_ = os.Setenv(key, prev)
		})
	}
}

func TestApplyDefaultProxyForNewChannelFromEnv(t *testing.T) {
	truncateTables(t)
	setProxyEnv(t, "http://env-proxy:7890")

	channel := &Channel{
		Type:   57,
		Name:   "env-proxy-channel",
		Key:    `{"access_token":"test","account_id":"acct"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "codex-plus",
	}

	changed := ApplyDefaultProxyForNewChannel(channel)
	require.True(t, changed)
	require.Equal(t, "http://env-proxy:7890", channel.GetSetting().Proxy)
}

func TestApplyDefaultProxyForNewChannelFromSameType(t *testing.T) {
	truncateTables(t)
	setProxyEnv(t, "")

	existing := &Channel{
		Type:   57,
		Name:   "codex-existing",
		Key:    `{"access_token":"test","account_id":"acct"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "codex-plus",
	}
	existing.SetSetting(dto.ChannelSettings{Proxy: "http://same-type-proxy:7890"})
	require.NoError(t, DB.Create(existing).Error)

	channel := &Channel{
		Type:   57,
		Name:   "codex-new",
		Key:    `{"access_token":"test2","account_id":"acct2"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "codex-pro20x",
	}

	changed := ApplyDefaultProxyForNewChannel(channel)
	require.True(t, changed)
	require.Equal(t, "http://same-type-proxy:7890", channel.GetSetting().Proxy)
}

func TestApplyDefaultProxyForNewChannelFromSharedGroup(t *testing.T) {
	truncateTables(t)
	setProxyEnv(t, "")

	existing := &Channel{
		Type:   1,
		Name:   "group-existing",
		Key:    "sk-test",
		Status: 1,
		Models: "gpt-4o-mini",
		Group:  "shared-group",
	}
	existing.SetSetting(dto.ChannelSettings{Proxy: "http://shared-group-proxy:7890"})
	require.NoError(t, DB.Create(existing).Error)

	channel := &Channel{
		Type:   57,
		Name:   "group-new",
		Key:    `{"access_token":"test3","account_id":"acct3"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "shared-group",
	}

	changed := ApplyDefaultProxyForNewChannel(channel)
	require.True(t, changed)
	require.Equal(t, "http://shared-group-proxy:7890", channel.GetSetting().Proxy)
}

func TestBatchInsertChannelsAppliesDefaultProxy(t *testing.T) {
	truncateTables(t)
	setProxyEnv(t, "")

	existing := Channel{
		Type:   57,
		Name:   "seed-channel",
		Key:    `{"access_token":"seed","account_id":"acct-seed"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "codex-plus",
	}
	existing.SetSetting(dto.ChannelSettings{Proxy: "http://seed-proxy:7890"})
	require.NoError(t, BatchInsertChannels([]Channel{existing}))

	inserted := Channel{
		Type:   57,
		Name:   "inserted-channel",
		Key:    `{"access_token":"inserted","account_id":"acct-inserted"}`,
		Status: 1,
		Models: "gpt-5.4",
		Group:  "codex-pro20x",
	}
	require.NoError(t, BatchInsertChannels([]Channel{inserted}))

	var loaded Channel
	require.NoError(t, DB.Where("name = ?", "inserted-channel").First(&loaded).Error)
	require.Equal(t, "http://seed-proxy:7890", loaded.GetSetting().Proxy)

	var abilities []Ability
	require.NoError(t, DB.Where("channel_id = ?", loaded.Id).Find(&abilities).Error)
	require.Len(t, abilities, 1)
}
