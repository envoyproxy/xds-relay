package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidHTTPMethod(t *testing.T) {
	req, err := http.NewRequest("PUT", "/ready/true", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := readyHandler()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestDefaultReadyStatus(t *testing.T) {
	req, err := http.NewRequest("GET", "/ready", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := readyHandler()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestReadyOverride(t *testing.T) {
	req, err := http.NewRequest("POST", "/ready/false", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := readyHandler()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req, err = http.NewRequest("GET", "/ready", nil)
	assert.NoError(t, err)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestMalformedReady(t *testing.T) {
	req, err := http.NewRequest("POST", "/ready/malformed", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := readyHandler()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	req, err = http.NewRequest("POST", "/ready/", nil)
	assert.NoError(t, err)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
