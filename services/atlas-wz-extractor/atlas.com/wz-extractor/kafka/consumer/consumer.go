package consumer

import (
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

// NewConfig is a curried builder mirroring atlas-data's. It looks up the topic
// name from the env var `token` and produces a consumer.Config.
func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

// LookupBrokers reads the cluster bootstrap servers from BOOTSTRAP_SERVERS.
func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
