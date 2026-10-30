// Copyright Contributors to the Open Cluster Management project

package send

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/reconciler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
)

func configureSendLoopTest(t *testing.T) {
	t.Helper()
	oldCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = oldCfg })
	config.Cfg.ValidationMaxPendingMS = config.DEFAULT_VALIDATION_MAX_PENDING_MS
	config.Cfg.ValidationRateMS = 15 * 60 * 1000
	config.Cfg.MaxBackoffMS = 8000
	// rand.Int with an upper bound of 1 produces zero jitter for deterministic timing.
	config.Cfg.RetryJitterMS = 1
}

func sendLoopResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newSendLoopTestSender(transport roundTripFunc) *Sender {
	return &Sender{
		aggregatorURL:            "https://indexer",
		aggregatorSyncPath:       "/sync",
		aggregatorValidationPath: "/validation",
		httpClient:               http.Client{Transport: transport},
		lastSentTime:             -1,
		rec:                      &reconciler.Reconciler{},
	}
}

func TestSendLoopValidationInterval(t *testing.T) {
	configureSendLoopTest(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()
		start := time.Now()
		var validationTimes []time.Duration
		postCount := 0
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postCount++
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			if len(validationTimes) == 3 {
				cancel()
			}
			return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
		})

		s.StartSendLoop(ctx)

		assert.Equal(t, []time.Duration{0, 15 * time.Minute, 30 * time.Minute}, validationTimes)
		assert.Equal(t, 1, postCount, "empty diffs must not send heartbeat POSTs")
		assert.Equal(t, start.Add(30*time.Minute), s.lastValidationTime)
	})
}

func TestSendLoopValidationFailuresUseIndependentBackoff(t *testing.T) {
	configureSendLoopTest(t)
	tests := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{name: "busy", status: http.StatusTooManyRequests, body: "busy"},
		{name: "database error", status: http.StatusInternalServerError, body: "unavailable"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: "unauthorized"},
		{name: "missing endpoint", status: http.StatusNotFound, body: "not found"},
		{name: "decode error", status: http.StatusOK, body: "not-json"},
		{name: "transport error", err: errors.New("connection failed")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				start := time.Now()
				lastValidation := start.Add(-15 * time.Minute)
				var validationTimes []time.Duration
				postCount := 0
				s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPost {
						postCount++
						return sendLoopResponse(http.StatusOK, `{}`), nil
					}
					validationTimes = append(validationTimes, time.Since(start))
					if len(validationTimes) == 5 {
						cancel()
					}
					if test.err != nil {
						return nil, test.err
					}
					return sendLoopResponse(test.status, test.body), nil
				})
				s.lastValidationTime = lastValidation

				s.StartSendLoop(ctx)

				assert.Equal(t, []time.Duration{0, 2 * time.Second, 6 * time.Second, 14 * time.Second, 22 * time.Second}, validationTimes)
				assert.Equal(t, 1, postCount, "pending validation must not force a full resync before the timeout")
				assert.Equal(t, start.Unix(), s.lastSentTime)
				assert.Equal(t, lastValidation, s.lastValidationTime, "failed attempts must not count as completed validation")
			})
		})
	}
}

func TestSendLoopValidationSuccessResetsBackoff(t *testing.T) {
	configureSendLoopTest(t)
	config.Cfg.ValidationRateMS = 10000
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		var validationTimes []time.Duration
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			switch len(validationTimes) {
			case 4:
				return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
			case 6:
				cancel()
				return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
			default:
				return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
			}
		})

		s.StartSendLoop(ctx)

		assert.Equal(t, []time.Duration{0, 2 * time.Second, 6 * time.Second, 14 * time.Second, 24 * time.Second, 26 * time.Second}, validationTimes)
		assert.Equal(t, start.Add(26*time.Second), s.lastValidationTime)
	})
}

func TestSendLoopValidationUsesCurrentCycleTotals(t *testing.T) {
	configureSendLoopTest(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		postCount, getCount := 0, 0
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postCount++
				// This send's adjusted expectation is one resource and one edge.
				return sendLoopResponse(http.StatusOK, `{"DeleteErrors":[{"ResourceUID":"stale"}],"DeleteEdgeErrors":[{"ResourceUID":"stale"}]}`), nil
			}
			getCount++
			if getCount == 1 {
				return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
			}
			cancel()
			// The later empty-diff cycle returns zero totals. Do not reuse the failed attempt's expectations.
			return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
		})

		s.StartSendLoop(ctx)

		assert.Equal(t, 1, postCount)
		assert.Equal(t, 2, getCount)
		assert.NotEqual(t, int64(-1), s.lastSentTime, "validation must use the current cycle's expected totals")
	})
}

func TestSendLoopSendsWhileValidationPending(t *testing.T) {
	configureSendLoopTest(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		var postTimes, validationTimes []time.Duration
		var s *Sender
		s = newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postTimes = append(postTimes, time.Since(start))
				if len(postTimes) == 2 {
					cancel()
				}
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			if len(validationTimes) == 2 {
				// Arrange another complete send while the next validation retry will wait until 6s.
				// The transport runs in the send-loop goroutine, so this does not race with Sync.
				s.lastSentTime = -1
			}
			return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
		})

		s.StartSendLoop(ctx)

		assert.Equal(t, []time.Duration{0, 4 * time.Second}, postTimes)
		assert.Equal(t, []time.Duration{0, 2 * time.Second}, validationTimes)
	})
}

func TestSendLoopValidationMismatchRequestsResync(t *testing.T) {
	configureSendLoopTest(t)
	for _, response := range []string{
		`{"TotalResources":1,"TotalEdges":0}`,
		`{"TotalResources":0,"TotalEdges":1}`,
	} {
		t.Run(response, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				start := time.Now()
				var postTimes []time.Duration
				getCount := 0
				s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPost {
						assert.Equal(t, "true", req.Header.Get("X-Overwrite-State"))
						postTimes = append(postTimes, time.Since(start))
						if len(postTimes) == 2 {
							cancel()
						}
						return sendLoopResponse(http.StatusOK, `{}`), nil
					}
					getCount++
					return sendLoopResponse(http.StatusOK, response), nil
				})

				s.StartSendLoop(ctx)

				assert.Equal(t, []time.Duration{0, 2 * time.Second}, postTimes, "mismatch must not increase send backoff")
				assert.Equal(t, 1, getCount)
			})
		})
	}
}

func TestSendLoopSerializesSendAndValidation(t *testing.T) {
	configureSendLoopTest(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		postStarted := make(chan struct{})
		releasePost := make(chan struct{})
		validationStarted := false
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				close(postStarted)
				<-releasePost
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationStarted = true
			cancel()
			return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
		})
		loopDone := make(chan struct{})
		go func() {
			s.StartSendLoop(ctx)
			close(loopDone)
		}()

		<-postStarted
		synctest.Wait()
		assert.False(t, validationStarted, "validation must wait for the send response")
		close(releasePost)
		<-loopDone
		assert.True(t, validationStarted)
	})
}

func TestSendLoopSkipsValidationAfterSendFailure(t *testing.T) {
	configureSendLoopTest(t)
	t.Setenv("DEPLOYED_IN_HUB", "true")
	t.Setenv("HUB_CONFIG", "")
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		getCount := 0
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				getCount++
			}
			cancel()
			return sendLoopResponse(http.StatusInternalServerError, "send failed"), nil
		})

		s.StartSendLoop(ctx)

		assert.Zero(t, getCount)
		assert.Equal(t, int64(-1), s.lastSentTime)
		assert.True(t, s.lastValidationTime.IsZero())
	})
}

func TestSendLoopValidationOverdueRequestsResyncAtCap(t *testing.T) {
	configureSendLoopTest(t)
	logState := klog.CaptureState()
	t.Cleanup(logState.Restore)
	var logs bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)
	klog.SetOutputBySeverity("WARNING", &logs)

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		start := time.Now()
		var validationTimes, postTimes []time.Duration
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postTimes = append(postTimes, time.Since(start))
				assert.Equal(t, "true", req.Header.Get("X-Overwrite-State"))
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			if time.Since(start) > time.Hour+20*time.Second {
				cancel()
			}
			return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
		})

		s.StartSendLoop(ctx)

		require.Greater(t, len(validationTimes), 100, "exercise repeated retries well past the backoff cap")
		for i := 3; i < len(validationTimes); i++ {
			assert.Equal(t, 8*time.Second, validationTimes[i]-validationTimes[i-1])
		}
		assert.Greater(t, validationTimes[len(validationTimes)-1], time.Hour)
		assert.Equal(t, []time.Duration{0, time.Hour + 2*time.Second}, postTimes)
		assert.True(t, s.lastValidationTime.IsZero())
	})
	klog.Flush()
	assert.Equal(t, 1, strings.Count(logs.String(), "Validation has been pending for"))
	assert.Contains(t, logs.String(), "Requesting complete sync.")
}

func TestSendLoopValidationMaxPendingOverride(t *testing.T) {
	configureSendLoopTest(t)
	config.Cfg.ValidationMaxPendingMS = 10000
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		var validationTimes, postTimes []time.Duration
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postTimes = append(postTimes, time.Since(start))
				assert.Equal(t, "true", req.Header.Get("X-Overwrite-State"))
				if len(postTimes) == 3 {
					cancel()
				}
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			return sendLoopResponse(http.StatusInternalServerError, "unavailable"), nil
		})

		s.StartSendLoop(ctx)

		// The timeout at 10s must be checked during the retry wait from 6s to 14s.
		// The new grace period prevents resyncing every 2s after the first forced resync.
		assert.Equal(t, []time.Duration{0, 12 * time.Second, 22 * time.Second}, postTimes)
		assert.Equal(t, []time.Duration{0, 2 * time.Second, 6 * time.Second, 14 * time.Second, 22 * time.Second}, validationTimes)
		assert.True(t, s.lastValidationTime.IsZero(), "a full sync must not count as successful validation")
	})
}

func TestSendLoopValidationSuccessRestartsResyncGracePeriod(t *testing.T) {
	configureSendLoopTest(t)
	config.Cfg.ValidationMaxPendingMS = 10000
	config.Cfg.ValidationRateMS = 4000
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		var validationTimes, postTimes []time.Duration
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postTimes = append(postTimes, time.Since(start))
				if len(postTimes) == 2 {
					cancel()
				}
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			validationTimes = append(validationTimes, time.Since(start))
			if len(validationTimes) == 2 {
				return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
			}
			return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
		})

		s.StartSendLoop(ctx)

		// Success at 2s starts the 10s window, including the normal validation interval.
		assert.Equal(t, []time.Duration{0, 14 * time.Second}, postTimes)
		assert.Equal(t, []time.Duration{0, 2 * time.Second, 6 * time.Second, 8 * time.Second, 12 * time.Second}, validationTimes)
		assert.Equal(t, start.Add(2*time.Second), s.lastValidationTime)
	})
}

func TestSendLoopValidationSuccessAtResyncDeadline(t *testing.T) {
	configureSendLoopTest(t)
	config.Cfg.ValidationMaxPendingMS = 10000
	config.Cfg.ValidationRateMS = 4000
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		postCount, getCount := 0, 0
		s := newSendLoopTestSender(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				postCount++
				return sendLoopResponse(http.StatusOK, `{}`), nil
			}
			getCount++
			if getCount == 1 || time.Since(start) == 10*time.Second {
				if getCount > 1 {
					cancel()
				}
				return sendLoopResponse(http.StatusOK, `{"TotalResources":0,"TotalEdges":0}`), nil
			}
			return sendLoopResponse(http.StatusTooManyRequests, "busy"), nil
		})

		s.StartSendLoop(ctx)

		assert.Equal(t, 1, postCount)
		assert.Equal(t, 4, getCount)
		assert.NotEqual(t, int64(-1), s.lastSentTime)
		assert.Equal(t, start.Add(10*time.Second), s.lastValidationTime)
	})
}
