package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// ShutdownHelper manages the graceful shutdown of various resources
type ShutdownHelper struct {
	log     logrus.FieldLogger
	timeout time.Duration
}

// NewShutdownHelper creates a new helper with a default 10s timeout.
// log accepts either a logger.Logger or a *logrus.Logger.
func NewShutdownHelper(log logrus.FieldLogger) *ShutdownHelper {
	return &ShutdownHelper{
		log:     log,
		timeout: 10 * time.Second,
	}
}

// WithTimeout overrides the total shutdown timeout.
func (h *ShutdownHelper) WithTimeout(timeout time.Duration) *ShutdownHelper {
	if timeout > 0 {
		h.timeout = timeout
	}
	return h
}

// Wait blocks until a termination signal is received (default: SIGINT, SIGTERM)
func (h *ShutdownHelper) Wait(signals ...os.Signal) {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, signals...)
	s := <-quit
	h.log.Infof("Shutdown signal received: %v", s)
}

// Graceful executes shutdown functions with a timeout
func (h *ShutdownHelper) Graceful(funcs map[string]func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	for name, f := range funcs {
		h.log.Infof("Shutting down %s...", name)
		if err := f(ctx); err != nil {
			h.log.Errorf("%s shutdown error: %v", name, err)
		} else {
			h.log.Infof("%s shut down successfully", name)
		}
	}
	h.log.Info("All resources shut down")
}
