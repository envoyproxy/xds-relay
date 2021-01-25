package util

import (
	"time"
)

// StringToDuration parses the string into a time.Duration value. If the string is empty, the
// default time.Duration is returned.
func StringToDuration(s string, defaultDuration time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultDuration, nil
	}
	return time.ParseDuration(s)
}
