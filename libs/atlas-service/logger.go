package service

import (
	"os"

	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

// CreateLogger is the fleet-canonical logger: stdout, ECS JSON formatting,
// a service.name field on every record, LOG_LEVEL env parsing (invalid
// values silently keep the default), and emit-time snake_case field-key
// normalization (see fieldnorm.go). The normalizer must stay the LAST
// registered hook so it sees keys added by earlier hooks.
func CreateLogger(serviceName string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.AddHook(newServiceNameHook(serviceName))
	l.SetFormatter(&ecslogrus.Formatter{})
	if val, ok := os.LookupEnv("LOG_LEVEL"); ok {
		if level, err := logrus.ParseLevel(val); err == nil {
			l.SetLevel(level)
		}
	}
	l.AddHook(fieldKeyNormalizerHook{})
	return l
}

type serviceNameHook struct {
	service string
}

func newServiceNameHook(name string) *serviceNameHook {
	return &serviceNameHook{service: name}
}

func (h *serviceNameHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *serviceNameHook) Fire(entry *logrus.Entry) error {
	entry.Data["service.name"] = h.service
	return nil
}
