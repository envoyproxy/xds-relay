package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

func TestShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blockedCh := make(chan bool, 2)
	l := &logger{}
	registerShutdownHandler(ctx, cancel, func() {
		blockedCh <- true
	},
		func(context.Context) error {
			blockedCh <- true
			return nil
		},
		l, time.Second*5)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-blockedCh
	<-blockedCh
}

func TestShutdownTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := &logger{blockedCh: make(chan bool, 1)}
	registerShutdownHandler(ctx, cancel, func() {
		<-time.After(time.Minute)
	}, func(context.Context) error { return nil }, l, time.Millisecond)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-l.blockedCh
	assert.Equal(t, "shutdown error: context deadline exceeded", l.lastErr)
}

func TestAdminServer_DefaultHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := defaultHandler()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "xds-relay admin API", rr.Body.String())
}

func TestAdminServer_NotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/not-implemented", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := defaultHandler()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())
}

type logger struct {
	blockedCh chan bool
	lastErr   string
}

func (l *logger) Named(name string) log.Logger {
	return l
}

func (l *logger) With(args ...interface{}) log.Logger {
	return l
}

func (l *logger) Sync() error { return nil }

func (l *logger) Debug(ctx context.Context, args ...interface{}) {
}

func (l *logger) Info(ctx context.Context, args ...interface{}) {
}

func (l *logger) Warn(ctx context.Context, args ...interface{}) {
}

func (l *logger) Error(ctx context.Context, args ...interface{}) {
	l.lastErr = fmt.Sprint(args...)
	l.blockedCh <- true
}

func (l *logger) Fatal(ctx context.Context, args ...interface{}) {
}

func (l *logger) Panic(ctx context.Context, args ...interface{}) {
}
