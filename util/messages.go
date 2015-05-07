package util

import (
	"git.totdev.in/totv/mika"
	"git.totdev.in/totv/mika/conf"
	log "github.com/Sirupsen/logrus"
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
	_, err := mika.RavenClient.CaptureMessage()
	if err != nil {
		log.Println("CaptureMessage: Failed to send message:", err)
	}
}
