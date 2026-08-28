package topic

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Provider model.Provider[string]

//goland:noinspection GoUnusedExportedFunction
func EnvProvider(l logrus.FieldLogger) func(token Token) Provider {
	return func(token Token) Provider {
		return func() (string, error) {
			t, ok := os.LookupEnv(string(token))
			if !ok || t == "" {
				return "", fmt.Errorf("topic token [%s] has no value in the environment", token)
			}
			return t, nil
		}
	}
}
