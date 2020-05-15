package integration

import (
	"context"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

// nolint
type gcpLogger struct {
	logger log.Logger
}

func (gcp gcpLogger) Debugf(format string, args ...interface{}) {
	gcp.logger.Debug(context.Background(), format, args)
}

func (gcp gcpLogger) Infof(format string, args ...interface{}) {
	gcp.logger.Info(context.Background(), format, args)
}

func (gcp gcpLogger) Warnf(format string, args ...interface{}) {
	gcp.logger.Warn(context.Background(), format, args)
}

func (gcp gcpLogger) Errorf(format string, args ...interface{}) {
	gcp.logger.Error(context.Background(), format, args)
}
