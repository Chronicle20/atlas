package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

type serviceNameHook struct {
	serviceName string
}

func newHook(serviceName string) *serviceNameHook {
	return &serviceNameHook{serviceName: serviceName}
}

func (h *serviceNameHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *serviceNameHook) Fire(entry *logrus.Entry) error {
	entry.Data["service.name"] = h.serviceName
	return nil
}

func CreateLogger(serviceName string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.AddHook(newHook(serviceName))
	fl := &ecslogrus.Formatter{}
	l.SetFormatter(fl)
	if val, ok := os.LookupEnv("LOG_LEVEL"); ok {
		if level, err := logrus.ParseLevel(val); err == nil {
			l.SetLevel(level)
		}
	}
	return l
}
