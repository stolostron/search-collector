// Copyright Contributors to the Open Cluster Management project

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLivenessProbe(t *testing.T) {
	// Given: a GET request against the /liveness endpoint
	req, reqErr := http.NewRequest("GET", "/liveness", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LivenessProbe)

	// When: we submit the request
	handler.ServeHTTP(rr, req)

	// Then: we get HTTP.StatusOK 200 without err
	assert.Nil(t, reqErr)
	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, rr.Body.String(), "OK")
}

func TestReadinessProbe(t *testing.T) {
	// Given: a GET request against the /readiness endpoint
	req, reqErr := http.NewRequest("GET", "/readiness", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LivenessProbe)

	// When: we submit the request
	handler.ServeHTTP(rr, req)

	// Then: we get HTTP.StatusOK 200 without err
	assert.Nil(t, reqErr)
	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, rr.Body.String(), "OK")
}
