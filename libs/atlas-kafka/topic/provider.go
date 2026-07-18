package topic

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Provider model.Provider[string]

//goland:noinspection GoUnusedExportedFunction
func EnvProvider(l logrus.FieldLogger) func(token string) Provider {
	return func(token string) Provider {
		return func() (string, error) {
			t, ok := os.LookupEnv(token)
			if !ok {
				l.Warnf("[%s] environment variable not set. Defaulting to provided token.", token)
				return token, nil
			}
			return t, nil
		}
	}
}
