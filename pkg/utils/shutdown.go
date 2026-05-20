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
	log     *logrus.Logger
	timeout time.Duration
}

// NewShutdownHelper creates a new helper with a default 10s timeout
func NewShutdownHelper(log *logrus.Logger) *ShutdownHelper {
	return &ShutdownHelper{
		log:     log,
		timeout: 10 * time.Second,
	}
}

// Wait blocks until a termination signal is received
func (h *ShutdownHelper) Wait() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
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
