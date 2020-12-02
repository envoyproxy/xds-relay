package log

import (
	"io"
	"os"
)

// MockLogger mocks a very basic debug logger.
var MockLogger = New("error", os.Stderr)

// NewMock returns an instance of Logger implemented using the Zap logging framework.
func NewMock(logLevel string, writeTo io.Writer) Logger {
	return New(logLevel, writeTo)
}
