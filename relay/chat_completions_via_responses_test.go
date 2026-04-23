package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestShouldForceResponsesUpstreamStream(t *testing.T) {
	if !shouldForceResponsesUpstreamStream(constant.ChannelTypeCodex, false) {
		t.Fatal("expected non-stream Codex compatibility request to force upstream streaming")
	}
	if shouldForceResponsesUpstreamStream(constant.ChannelTypeCodex, true) {
		t.Fatal("expected already-streaming Codex request to keep current behavior")
	}
	if shouldForceResponsesUpstreamStream(constant.ChannelTypeOpenAI, false) {
		t.Fatal("expected non-Codex channels to remain unchanged")
	}
}
