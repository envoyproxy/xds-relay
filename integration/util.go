package integration

import "log"

type gcpLogger struct{}

func (logger gcpLogger) Debugf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger gcpLogger) Infof(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger gcpLogger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger gcpLogger) Errorf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}
