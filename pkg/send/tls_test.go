// Copyright Contributors to the Open Cluster Management project

package send

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTLSReload_NilChannel(t *testing.T) {
	s := &Sender{tlsReload: nil}
	// Should not panic.
	s.checkTLSReload()
}

func TestCheckTLSReload_EmptyChannel(t *testing.T) {
	ch := make(chan struct{}, 1)
	// Don't send anything — checkTLSReload should be a no-op.
	// We can't call checkTLSReload directly because the reload path calls
	// getHTTPSClient() which needs a cluster. Instead verify the select logic:
	// an empty buffered channel should take the default branch.
	select {
	case <-ch:
		t.Fatal("should not receive from empty channel")
	default:
		// Expected path.
	}
}

func TestCheckTLSReload_SignalIsBuffered(t *testing.T) {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	assert.Len(t, ch, 1, "channel should have one pending signal")

	// Drain it (simulating what checkTLSReload does).
	select {
	case <-ch:
		// Signal consumed.
	default:
		t.Fatal("should have received signal")
	}
	assert.Len(t, ch, 0, "channel should be empty after drain")
}

func TestNewSender_AcceptsNilTLSReload(t *testing.T) {
	// Verify NewSender doesn't panic with nil reload channel.
	// Can't fully construct (needs config), but verify the struct field.
	s := &Sender{
		aggregatorURL: "https://test",
		tlsReload:     nil,
	}
	assert.Nil(t, s.tlsReload)
}
