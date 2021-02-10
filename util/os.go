package util

import (
	"context"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// WaitForSignal will execute a function when a matching os.Signal is received
// This is mostly designed to shutdown & cleanup services
func WaitForSignal(ctx context.Context, f func(ctx context.Context) error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan
	c, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
	defer cancel()
	if err := f(c); err != nil {
		log.Errorf("Error closing servers gracefully; %s", err)
	}
}

// Exists checks for the existence of a file path
func Exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

// FindFile will walk up the directory tree until it find a file. Max depth of 4
func FindFile(p string) string {
	var dots []string
	for i := 0; i < 4; i++ {
		dir := path.Join(dots...)
		fPath := path.Join(dir, p)
		if Exists(fPath) {
			fp, err := filepath.Abs(fPath)
			if err == nil {
				return fp
			}
			return fp
		}
		if strings.HasSuffix(dir, "mika") {
			return p
		}
		dots = append(dots, "..")
	}
	return p
}

func Now() time.Time {
	// TODO configure local or utc tz
	return time.Now().UTC()
}
