package server

import (
	"context"
	"fmt"
	"syscall"
	"testing"

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
		l)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-blockedCh
	<-blockedCh
}

type logger struct {
	blockedCh chan bool
	lastErr   string
}

func (l *logger) Named(name string) log.Logger                                    { return l }
func (l *logger) With(args ...interface{}) log.Logger                             { return l }
func (l *logger) Sync() error                                                     { return nil }
func (l *logger) Debug(ctx context.Context, template string, args ...interface{}) {}
func (l *logger) Info(ctx context.Context, template string, args ...interface{})  {}
func (l *logger) Warn(ctx context.Context, template string, args ...interface{})  {}
func (l *logger) Fatal(ctx context.Context, template string, args ...interface{}) {}
func (l *logger) Panic(ctx context.Context, template string, args ...interface{}) {}
func (l *logger) Error(ctx context.Context, template string, args ...interface{}) {
	l.lastErr = fmt.Sprintf(template+"%v", args...)
	l.blockedCh <- true
}
func (l *logger) UpdateLogLevel(logLevel string) {}
func (l *logger) GetLevel() string               { return "" }
