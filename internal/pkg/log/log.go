// Package log configures a logger using the Zap logging framework.
package log

import (
	"context"

	"github.com/envoyproxy/xds-relay/internal/pkg/log/zap"

	z "go.uber.org/zap"
)

// Logger is the contract for xds-relay's logging implementation.
//
// A self-contained usage example looks as follows:
//
//    Log.Named("foo-component").With(
//        "field1", "value1",
//        "field2", "value2",
//     ).Error("my error message")
type Logger interface {
	// Named adds a sub-scope to the logger.
	Named(name string) Logger

	// With adds a variadic number of fields to the logging context.
	// When processing pairs, the first element of the pair is used as the
	// field key and the second as the field value.
	//
	// For example,
	//
	//   Log.With(
	//     "hello", "world",
	//     "failure", errors.New("oh no"),
	//     "count", 42,
	//     "user", User{Name: "alice"},
	//  ).Info("this is an error message")
	With(args ...interface{}) Logger

	// Log a message at level Debug, annotated with fields provided through With().
	Debug(ctx context.Context, msg ...interface{})

	// Log a message at level Info, annotated with fields provided through With().
	Info(ctx context.Context, msg ...interface{})

	// Log a message at level Warn, annotated with fields provided through With().
	Warn(ctx context.Context, msg ...interface{})

	// Log a message at level Error, annotated with fields provided through With().
	Error(ctx context.Context, msg ...interface{})

	// Log a message at level Panic, annotated with fields provided through With(), and immediately
	// panic.
	Panic(ctx context.Context, msg ...interface{})

	// Log a message at level Fatal, annotated with fields provided through With(), and immediately
	// call os.Exit.
	Fatal(ctx context.Context, msg ...interface{})

	// Sync flushes any buffered log entries.
	Sync() error
}

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

func (l *logger) Debug(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Debug(args...)
}

func (l *logger) Info(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Info(args...)
}

func (l *logger) Warn(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Warn(args...)
}

func (l *logger) Error(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Error(args...)
}

func (l *logger) Fatal(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Fatal(args...)
}

func (l *logger) Panic(ctx context.Context, args ...interface{}) {
	l.WithContext(ctx).zap.Panic(args...)
}
