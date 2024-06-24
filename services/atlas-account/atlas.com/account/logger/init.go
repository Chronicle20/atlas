package logger

import (
	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
	"os"
)

func CreateLogger(serviceName string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.AddHook(newHook(serviceName))
	l.SetFormatter(&ecslogrus.Formatter{})
	if val, ok := os.LookupEnv("LOG_LEVEL"); ok {
		if level, err := logrus.ParseLevel(val); err == nil {
			l.SetLevel(level)
		}
	}
	return l
}

type ExtraFieldHook struct {
	service string
}

func newHook(name string) *ExtraFieldHook {
	return &ExtraFieldHook{service: name}
}

func (h *ExtraFieldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *ExtraFieldHook) Fire(entry *logrus.Entry) error {
	entry.Data["service.name"] = h.service
	return nil
}
