package util

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

// waitForTermination blocks until the system signals termination or done has a value
func WaitForTermination(log logrus.FieldLogger, done <-chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-signals:
		log.Infof("Triggering shutdown from signal %s", sig)
	case <-done:
		log.Infof("Shutting down...")
	}
}
