package log

import (
	"context"
	"io"

	"github.com/envoyproxy/xds-relay/internal/pkg/log/zap"
	z "go.uber.org/zap"
)

type logger struct {
	zap     *z.SugaredLogger
	writeTo io.Writer
	level   string
}

// New returns an instance of Logger implemented using the Zap logging framework.
//
// logLevel is mandatory, can be one of DEBUG, INFO, WARN, ERROR, FATAL.
// writeTo is mandatory. This is the writer where logs should be outputted to.
// Use os.Stderr if unsure.
func New(logLevel string, writeTo io.Writer) Logger {
	zLevel, parseLogLevelErr := zap.ParseLogLevel(logLevel)

	log := zap.New(
		zap.Level(&zLevel),
		// CallerSkip skips 1 number of callers, otherwise the file that gets
		// logged will always be the wrapped file. In this case, log.go.
		zap.AddCallerSkip(1),
		zap.WriteTo(writeTo),
	)

	if parseLogLevelErr != nil {
		// Log an invalid log level error and set the default level to info.
		log.Error("cannot set logger to desired log level")
	}

	log = log.With(z.Namespace("json"))
	return &logger{zap: log.Sugar(), writeTo: writeTo, level: zLevel.String()}
}

// UpdateLevel updates the logging level for the logger instance by
// internally creating a new logger at the new level.
func (l *logger) UpdateLogLevel(logLevel string) {
	zLevel, parseLogLevelErr := zap.ParseLogLevel(logLevel)

	log := zap.New(
		zap.Level(&zLevel),
		// CallerSkip skips 1 number of callers, otherwise the file that gets
		// logged will always be the wrapped file. In this case, log.go.
		zap.AddCallerSkip(1),
		zap.WriteTo(l.writeTo),
	)

	if parseLogLevelErr != nil {
		// Log an invalid log level error and set the default level to info.
		log.Error("cannot set logger to desired log level")
	}

	log = log.With(z.Namespace("json"))
	l.zap = log.Sugar()
	l.level = zLevel.String()
}

// GetLevel returns the logging level in human-readable string format..
func (l *logger) GetLevel() string {
	return l.level
}

func (l *logger) Named(name string) Logger {
	return &logger{zap: l.zap.Named(name)}
}

func (l *logger) With(args ...interface{}) Logger {
	return &logger{zap: l.zap.With(args...)}
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
