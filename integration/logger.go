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
	gcp.logger.Debugf(context.Background(), format, args)
}

func (gcp gcpLogger) Infof(format string, args ...interface{}) {
	gcp.logger.Infof(context.Background(), format, args)
}

func (gcp gcpLogger) Warnf(format string, args ...interface{}) {
	gcp.logger.Warnf(context.Background(), format, args)
}

func (gcp gcpLogger) Errorf(format string, args ...interface{}) {
	gcp.logger.Errorf(context.Background(), format, args)
}
