// Package zap sets up a logger using the Zap logging framework.
package zap

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options contains all possible log settings.
type Options struct {
	// CallerSkip
	CallerSkip int
	// Level configures the log verbosity. Defaults to Debug.
	Level *zap.AtomicLevel
	// StacktraceLevel is the level which stacktraces will be emitted. Defaults
	// to Warn.
	StacktraceLevel *zap.AtomicLevel
	// Encoder configures how Zap will encode the output. Defaults to JSON.
	Encoder zapcore.Encoder
	// OutputDest controls the destination of the log output. Defaults to
	// os.Stderr.
	OutputDest io.Writer
	// ZapOptions allows passing additional optional zap.Options, ex: Sampling.
	ZapOptions []zap.Option
}

// Opts allows manipulation of the Zap options.
type Opts func(*Options)

// addDefaults adds defaults to the Options
func (o *Options) addDefaults() {
	if o.CallerSkip < 1 {
		o.CallerSkip = 1
	}
	if o.OutputDest == nil {
		o.OutputDest = os.Stderr
	}
	if o.Encoder == nil {
		encCfg := zap.NewProductionEncoderConfig()
		o.Encoder = zapcore.NewJSONEncoder(encCfg)
	}
	if o.Level == nil {
		level := zap.NewAtomicLevelAt(zap.DebugLevel)
		o.Level = &level
	}
	if o.StacktraceLevel == nil {
		level := zap.NewAtomicLevelAt(zap.WarnLevel)
		o.StacktraceLevel = &level
	}

	o.ZapOptions = append(o.ZapOptions, zap.AddStacktrace(o.StacktraceLevel))
}

// New returns a new zap.Logger configured with the passed Options or their
// defaults.
func New(opts ...Opts) *zap.Logger {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	o.addDefaults()
	sink := zapcore.AddSync(o.OutputDest)
	o.ZapOptions = append(o.ZapOptions, zap.AddCallerSkip(o.CallerSkip), zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(o.Encoder, sink, *o.Level))
	log = log.WithOptions(o.ZapOptions...)
	return log
}

// AddCallerSkip increases the number of callers skipped by caller annotation,
// supplying this Option prevents zap from always reporting the wrapper code as
// the caller.
func AddCallerSkip(skip int) Opts {
	return func(o *Options) {
		o.CallerSkip = o.CallerSkip + skip
	}
}

// WriteTo configures the logger to write to the given io.Writer, instead of
// stderr. See Options.OutputDest.
func WriteTo(out io.Writer) Opts {
	return func(o *Options) {
		o.OutputDest = out
	}
}

// Encoder configures how the logger will encode the output e.g console, JSON.
// See Options.Encoder.
func Encoder(encoder zapcore.Encoder) Opts {
	return func(o *Options) {
		o.Encoder = encoder
	}
}

// Level sets the the minimum enabled logging level e.g Debug, Info, Warn,
// Error. See Options.Level.
func Level(level *zap.AtomicLevel) Opts {
	return func(o *Options) {
		o.Level = level
	}
}

// StacktraceLevel configures the logger to record a stack trace for all messages at
// or above the given level. See Options.StacktraceLevel.
func StacktraceLevel(stacktraceLevel *zap.AtomicLevel) Opts {
	return func(o *Options) {
		o.StacktraceLevel = stacktraceLevel
	}
}

// RawOptions allows appending additional zap.Options. See Options.ZapOptions.
func RawOptions(options ...zap.Option) Opts {
	return func(o *Options) {
		o.ZapOptions = append(o.ZapOptions, options...)
	}
}

// ParseLogLevel accepts either a capitalized or lower cased string for the log
// level. Accepts one of "debug", "info", "warn", "error", "panic", or "fatal".
// Returns zap.InfoLevel on an error.
func ParseLogLevel(logLevel string) (zap.AtomicLevel, error) {
	l := zap.NewAtomicLevel()
	return l, l.UnmarshalText([]byte(logLevel))
}
