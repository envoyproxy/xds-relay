package log

import (
	"os"
)

// MockLogger mocks a very basic debug logger.
var MockLogger = New("debug", os.Stderr)
