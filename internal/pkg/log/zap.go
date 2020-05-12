package log

import (
	"context"

	"github.com/envoyproxy/xds-relay/internal/pkg/log/zap"
	z "go.uber.org/zap"
)

type logger struct {
	zap *z.SugaredLogger
}

// New returns an instance of Logger implemented using the Zap logging framework.
func New(logLevel string) Logger {
	zLevel, parseLogLevelErr := zap.ParseLogLevel(logLevel)

	log := zap.New(
		zap.Level(&zLevel),
		// CallerSkip skips 1 number of callers, otherwise the file that gets
		// logged will always be the wrapped file. In this case, log.go.
		zap.AddCallerSkip(1),
	)

	if parseLogLevelErr != nil {
		// Log an invalid log level error and set the default level to info.
		log.Error("cannot set logger to desired log level")
	}
	return &logger{zap: log.Sugar()}
}

func (l *logger) Named(name string) Logger {
	l.zap = l.zap.Named(name)
	return l
}

func (l *logger) With(args ...interface{}) Logger {
	l.zap = l.zap.With(args...)
	return l
}

func (l *logger) WithContext(ctx context.Context) *logger {
	// We can add origin xDS request context here later.
	// For now, just return the logger.
	return l
}

func (l *logger) Sync() error { return l.zap.Sync() }

func (l *logger) Debug(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Debugf(template, args...)
}

func (l *logger) Info(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Infof(template, args...)
}

func (l *logger) Warn(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Warnf(template, args...)
}

func (l *logger) Error(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Errorf(template, args...)
}

func (l *logger) Fatal(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Fatalf(template, args...)
}

func (l *logger) Panic(ctx context.Context, template string, args ...interface{}) {
	l.WithContext(ctx).zap.Panicf(template, args...)
}
