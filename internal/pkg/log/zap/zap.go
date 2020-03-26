// Package zap sets up a logger using the Zap logging framework.
package zap

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// options contains all possible log settings.
type options struct {
	// callerSkip increases the number of callers skipped by caller annotation.
	callerSkip int
	// level configures the log verbosity. Defaults to Debug.
	level *zap.AtomicLevel
	// stacktraceLevel is the level which stacktraces will be emitted. Defaults
	// to Warn.
	stacktraceLevel *zap.AtomicLevel
	// encoder configures how Zap will encode the output. Defaults to JSON.
	encoder zapcore.Encoder
	// outputDest controls the destination of the log output. Defaults to
	// os.Stderr.
	outputDest io.Writer
	// zapOptions allows passing additional optional zap.Options, ex: Sampling.
	zapOptions []zap.Option
}

// Opts allows manipulation of the Zap options.
type Opts func(*options)

// addDefaults adds defaults to the Options
func (o *options) addDefaults() {
	if o.callerSkip < 1 {
		o.callerSkip = 1
	}
	if o.outputDest == nil {
		o.outputDest = os.Stderr
	}
	if o.encoder == nil {
		encCfg := zap.NewProductionEncoderConfig()
		o.encoder = zapcore.NewJSONEncoder(encCfg)
	}
	if o.level == nil {
		level := zap.NewAtomicLevelAt(zap.DebugLevel)
		o.level = &level
	}
	if o.stacktraceLevel == nil {
		level := zap.NewAtomicLevelAt(zap.WarnLevel)
		o.stacktraceLevel = &level
	}

	o.zapOptions = append(o.zapOptions, zap.AddStacktrace(o.stacktraceLevel))
}

// New returns a new zap.Logger configured with the passed Options or their
// defaults.
func New(opts ...Opts) *zap.Logger {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	o.addDefaults()
	sink := zapcore.AddSync(o.outputDest)
	o.zapOptions = append(o.zapOptions, zap.AddCallerSkip(o.callerSkip), zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(o.encoder, sink, *o.level))
	log = log.WithOptions(o.zapOptions...)
	return log
}

// AddCallerSkip increases the number of callers skipped by caller annotation,
// supplying this Option prevents zap from always reporting the wrapper code as
// the caller.
func AddCallerSkip(skip int) Opts {
	return func(o *options) {
		o.callerSkip = o.callerSkip + skip
	}
}

// WriteTo configures the logger to write to the given io.Writer, instead of
// stderr. See Options.OutputDest.
func WriteTo(out io.Writer) Opts {
	return func(o *options) {
		o.outputDest = out
	}
}

// Encoder configures how the logger will encode the output e.g console, JSON.
// See Options.Encoder.
func Encoder(encoder zapcore.Encoder) Opts {
	return func(o *options) {
		o.encoder = encoder
	}
}

// Level sets the the minimum enabled logging level e.g Debug, Info, Warn,
// Error. See Options.Level.
func Level(level *zap.AtomicLevel) Opts {
	return func(o *options) {
		o.level = level
	}
}

// StacktraceLevel configures the logger to record a stack trace for all messages at
// or above the given level. See Options.StacktraceLevel.
func StacktraceLevel(stacktraceLevel *zap.AtomicLevel) Opts {
	return func(o *options) {
		o.stacktraceLevel = stacktraceLevel
	}
}

// RawOptions allows appending additional zap.Options. See Options.ZapOptions.
func RawOptions(opts ...zap.Option) Opts {
	return func(o *options) {
		o.zapOptions = append(o.zapOptions, opts...)
	}
}

// ParseLogLevel accepts either a capitalized or lower cased string for the log
// level. Accepts one of "debug", "info", "warn", "error", "panic", or "fatal".
// Returns zap.InfoLevel on an error.
func ParseLogLevel(logLevel string) (zap.AtomicLevel, error) {
	l := zap.NewAtomicLevel()
	return l, l.UnmarshalText([]byte(logLevel))
}
