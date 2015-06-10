package util

import (
	"git.totdev.in/totv/mika/conf"
	"strings"
)

func CaptureMessage(message ...string) {
	if conf.Config.SentryDSN == "" {
		return
	}
	msg := strings.Join(message, "")
	if msg == "" {
		return
	}
}
