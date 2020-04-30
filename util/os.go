package util

import (
	"context"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// WaitForSignal will execute a function when a matching os.Signal is received
// This is mostly designed to shutdown & cleanup services
func WaitForSignal(ctx context.Context, f func(ctx context.Context) error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-sigChan:
		c, _ := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
		if err := f(c); err != nil {
			log.Fatalf("Error closing servers gracefully; %s", err)
		}
	}
}
