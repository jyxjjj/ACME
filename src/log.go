package main

import (
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func initLogger() {
	Log = logrus.New()
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
		FullTimestamp:             true,
	})
}
