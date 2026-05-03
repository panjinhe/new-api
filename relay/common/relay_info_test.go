package common

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoBeginUpstreamRequestResetsAttemptTiming(t *testing.T) {
	first := time.Unix(100, 0)
	second := first.Add(time.Second)
	info := &RelayInfo{}

	info.BeginUpstreamRequest(first)
	info.SetUpstreamRequestWroteTime(first.Add(10 * time.Millisecond))
	info.SetUpstreamFirstByteTime(first.Add(20 * time.Millisecond))
	info.SetUpstreamResponseHeaderTime(first.Add(30 * time.Millisecond))
	info.SetUpstreamGotConnTime(first.Add(5*time.Millisecond), true)

	info.BeginUpstreamRequest(second)

	require.Equal(t, second, info.UpstreamRequestStartTime)
	require.True(t, info.UpstreamRequestWroteTime.IsZero())
	require.True(t, info.UpstreamFirstByteTime.IsZero())
	require.True(t, info.UpstreamResponseHeaderTime.IsZero())
	require.True(t, info.UpstreamGotConnTime.IsZero())
	require.False(t, info.UpstreamReusedConn)
}

func TestRelayInfoGatewayStageDurationsAccumulate(t *testing.T) {
	info := &RelayInfo{}

	info.AddGatewayStageDuration("token_count", 150*time.Millisecond)
	info.AddGatewayStageDuration("token_count", 25*time.Millisecond)
	info.AddGatewayStageDuration(" ", 500*time.Millisecond)
	info.AddGatewayStageDuration("ignored", -time.Millisecond)

	require.EqualValues(t, 175, info.GatewayStageTimings["token_count"])
	_, ok := info.GatewayStageTimings["ignored"]
	require.False(t, ok)
}
