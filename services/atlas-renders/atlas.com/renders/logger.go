package main

import "github.com/sirupsen/logrus"

func newLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	return l
}
