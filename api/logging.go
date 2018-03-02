package api

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/drasko/go-auth-kit/http"
	"github.com/drasko/go-auth-kit/writer"
)

var _ auth.Service = (*loggingService)(nil)

type loggingService struct {
	logger log.Logger
	auth.Service
}

// NewLoggingService adds logging facilities to the adapter.
func NewLoggingService(logger log.Logger, s auth.Service) auth.Service {
	return &loggingService{logger, s}
}

func (ls *loggingService) Publish(msg writer.RawMessage) error {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "publish",
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.Publish(msg)
}
