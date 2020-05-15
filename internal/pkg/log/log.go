// Package log defines the contract for the xds-relay logger.
// It also contains an implementation of the contract using the Zap logging
// framework.
package log

import (
	"context"
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

	// Debug logs a message at level Debug with support for string formatting, annotated with
	// fields provided through With().
	Debug(ctx context.Context, template string, msg ...interface{})

	// Info logs a message at level Info with support for string formatting, annotated with fields
	// provided through With().
	Info(ctx context.Context, template string, msg ...interface{})

	// Warn logs a message at level Warn with support for string formatting, annotated with fields
	// provided through With().
	Warn(ctx context.Context, template string, msg ...interface{})

	// Error logs a message at level Error with support for string formatting, annotated with
	// fields provided through With().
	Error(ctx context.Context, template string, msg ...interface{})

	// Panic logs a message at level Panic with support for string formatting, annotated with
	// fields provided through With(), and immediately panics.
	Panic(ctx context.Context, template string, msg ...interface{})

	// Fatal logs a message at level Fatal with support for string formatting, annotated with
	// fields provided through With(), and immediately calls os.Exit.
	Fatal(ctx context.Context, template string, msg ...interface{})

	// Sync flushes any buffered log entries.
	Sync() error
}
