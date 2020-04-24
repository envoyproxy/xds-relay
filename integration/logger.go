package integration

import (
	"context"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

type gcpLogger struct {
	logger log.Logger
}

func (logger gcpLogger) Debugf(format string, args ...interface{}) {
	logger.logger.Debug(context.Background(), format, args)
}

func (logger gcpLogger) Infof(format string, args ...interface{}) {
	logger.logger.Info(context.Background(), format, args)
}

func (logger gcpLogger) Warnf(format string, args ...interface{}) {
	logger.logger.Warn(context.Background(), format, args)
}

func (logger gcpLogger) Errorf(format string, args ...interface{}) {
	logger.logger.Error(context.Background(), format, args)
}
