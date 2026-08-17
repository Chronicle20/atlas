package service

import (
	"os"

	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// CreateLogger is the fleet-canonical logger: stdout, ECS JSON formatting,
// a service.name field on every record, an environment field sourced from
// env.Self() (FR-10, empty/omitted on main for byte-identical output,
// NFR-7), LOG_LEVEL env parsing (invalid values silently keep the default),
// and emit-time snake_case field-key normalization (see fieldnorm.go). The
// normalizer must stay the LAST registered hook so it sees keys added by
// earlier hooks.
func CreateLogger(serviceName string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.AddHook(newServiceNameHook(serviceName))
	l.AddHook(environmentHook{environment: string(env.Self())})
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

// environmentHook stamps the environment field (FR-10) sourced from
// env.Self(), which is process-local (ATLAS_ENVIRONMENT) and trusted —
// never from a message or request header. On main, environment is "" and
// the hook is a no-op so log records stay byte-identical to today's
// (NFR-7). A record whose context already carries a different environment
// keeps it: the hook fills in, it does not overwrite.
type environmentHook struct{ environment string }

func (h environmentHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h environmentHook) Fire(entry *logrus.Entry) error {
	if h.environment == "" {
		return nil // main: byte-identical to today's records (NFR-7)
	}
	if _, present := entry.Data["environment"]; !present {
		entry.Data["environment"] = h.environment
	}
	return nil
}
