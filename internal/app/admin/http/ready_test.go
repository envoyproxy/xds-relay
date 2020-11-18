package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidHTTPMethod(t *testing.T) {
	handler := readyHandler(make(chan<- bool))

	verifyState(t, handler, "/ready/true", http.StatusMethodNotAllowed, http.MethodPut)
}

func TestDefaultReadyStatus(t *testing.T) {
	handler := readyHandler(make(chan<- bool))

	verifyState(t, handler, "/ready", http.StatusOK, http.MethodGet)
}

func TestReadyOverride(t *testing.T) {
	ch := make(chan bool, 1)
	handler := readyHandler(ch)

	verifyState(t, handler, "/ready/true", http.StatusOK, http.MethodPost)
	select {
	case <-ch:
		assert.Fail(t, "Same state override should not happen")
	default:
	}

	verifyState(t, handler, "/ready/false", http.StatusOK, http.MethodPost)
	change := <-ch
	assert.False(t, change)

	verifyState(t, handler, "/ready/false", http.StatusOK, http.MethodPost)
	select {
	case <-ch:
		assert.Fail(t, "Same state override should not happen")
	default:
	}
}

func TestMalformedReady(t *testing.T) {
	handler := readyHandler(make(chan<- bool))

	verifyState(t, handler, "/ready/malformed", http.StatusInternalServerError, http.MethodPost)
	verifyState(t, handler, "/ready/", http.StatusInternalServerError, http.MethodPost)
}

func TestRecipientNotReadyReturnsError(t *testing.T) {
	// Use a unbuffered channel with no listener to simulate blocking
	handler := readyHandler(make(chan<- bool))

	verifyState(t, handler, "/ready/false", http.StatusInternalServerError, http.MethodPost)
}

func verifyState(t *testing.T, handler http.HandlerFunc, url string, code int, method string) {
	req, err := http.NewRequest(method, url, nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, code, rr.Code)
}
