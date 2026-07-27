// Copyright Contributors to the Open Cluster Management project

package config

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeConfigMapGetter is a test double for configMapGetter.
type fakeConfigMapGetter struct {
	mu      sync.Mutex
	results []fakeGetResult
	call    int
}

type fakeGetResult struct {
	cm  *corev1.ConfigMap
	err error
}

func (f *fakeConfigMapGetter) Get(ctx context.Context, name, namespace string) (*corev1.ConfigMap, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.call >= len(f.results) {
		// Return the last result for any extra calls.
		r := f.results[len(f.results)-1]
		return r.cm, r.err
	}
	r := f.results[f.call]
	f.call++
	return r.cm, r.err
}

func newConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: tlsProfileConfigMap, Namespace: "test-ns"},
		Data:       data,
	}
}

func TestPollTLSProfile_ChangeSignalsReload(t *testing.T) {
	data1 := map[string]string{
		"profileType":   "Intermediate",
		"minTLSVersion": "VersionTLS12",
		"cipherSuites":  "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	}
	data2 := map[string]string{
		"profileType":   "Modern",
		"minTLSVersion": "VersionTLS13",
		"cipherSuites":  "TLS_AES_128_GCM_SHA256",
	}

	getter := &fakeConfigMapGetter{
		results: []fakeGetResult{
			{cm: newConfigMap(data1), err: nil}, // initial read
			{cm: newConfigMap(data1), err: nil}, // first tick — same data
			{cm: newConfigMap(data2), err: nil}, // second tick — changed
		},
	}

	reload := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollTLSProfile(ctx, reload, getter, 10*time.Millisecond, "test-ns")

	// Should get a reload signal when data changes.
	select {
	case <-reload:
		// Expected.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reload signal when ConfigMap data changes")
	}
}

func TestPollTLSProfile_NoSignalWhenUnchanged(t *testing.T) {
	data := map[string]string{
		"profileType":   "Intermediate",
		"minTLSVersion": "VersionTLS12",
		"cipherSuites":  "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	}

	getter := &fakeConfigMapGetter{
		results: []fakeGetResult{
			{cm: newConfigMap(data), err: nil}, // initial read
			{cm: newConfigMap(data), err: nil}, // tick — same data
			{cm: newConfigMap(data), err: nil}, // tick — same data
		},
	}

	reload := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollTLSProfile(ctx, reload, getter, 10*time.Millisecond, "test-ns")

	// Wait long enough for several ticks.
	time.Sleep(100 * time.Millisecond)

	select {
	case <-reload:
		t.Fatal("should not signal when data is unchanged")
	default:
		// Expected.
	}
}

func TestPollTLSProfile_InitialErrorThenAppears(t *testing.T) {
	data := map[string]string{
		"profileType":   "Intermediate",
		"minTLSVersion": "VersionTLS12",
		"cipherSuites":  "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	}

	getter := &fakeConfigMapGetter{
		results: []fakeGetResult{
			{cm: nil, err: fmt.Errorf("not found")},  // initial read fails
			{cm: nil, err: fmt.Errorf("not found")},   // first tick — still failing
			{cm: newConfigMap(data), err: nil},         // second tick — appears
		},
	}

	reload := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollTLSProfile(ctx, reload, getter, 10*time.Millisecond, "test-ns")

	// nil != non-nil data should trigger a signal.
	select {
	case <-reload:
		// Expected.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reload signal when ConfigMap appears after initial failure")
	}
}

func TestPollTLSProfile_ContextCancellationStops(t *testing.T) {
	data := map[string]string{"profileType": "Intermediate"}

	getter := &fakeConfigMapGetter{
		results: []fakeGetResult{
			{cm: newConfigMap(data), err: nil},
		},
	}

	reload := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		pollTLSProfile(ctx, reload, getter, 10*time.Millisecond, "test-ns")
		close(done)
	}()

	// Cancel and verify the poller exits.
	cancel()
	select {
	case <-done:
		// Expected: poller returned.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("poller did not stop after context cancellation")
	}
}

func TestPollTLSProfile_GetErrorDuringPollContinues(t *testing.T) {
	data1 := map[string]string{
		"profileType":   "Intermediate",
		"minTLSVersion": "VersionTLS12",
	}
	data2 := map[string]string{
		"profileType":   "Modern",
		"minTLSVersion": "VersionTLS13",
	}

	// Sequence: data1 → error → data1 → data2
	// The data1 after the error must NOT trigger a reload (proves lastData is preserved).
	// Only the data2 should trigger exactly one reload.
	getter := &fakeConfigMapGetter{
		results: []fakeGetResult{
			{cm: newConfigMap(data1), err: nil},       // initial read
			{cm: nil, err: fmt.Errorf("transient")},   // tick 1 — error, should continue
			{cm: newConfigMap(data1), err: nil},        // tick 2 — same data as before error, no signal
			{cm: newConfigMap(data2), err: nil},        // tick 3 — changed data, signal
		},
	}

	reload := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollTLSProfile(ctx, reload, getter, 10*time.Millisecond, "test-ns")

	// Should detect the change only when data2 appears (not when data1 returns after error).
	select {
	case <-reload:
		// Expected: one signal for data1→data2.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reload signal after transient error recovery")
	}

	// Verify no extra signals — if lastData had been reset to nil during the error,
	// the data1 return would have also triggered a spurious signal.
	assert.Len(t, reload, 0, "should have exactly one signal, no extras")
}
