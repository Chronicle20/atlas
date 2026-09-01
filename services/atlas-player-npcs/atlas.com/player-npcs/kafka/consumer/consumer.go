// Package consumer is the service-local wiring wrapper around
// libs/atlas-kafka/consumer's Config -- copied from
// services/atlas-notes/atlas.com/notes/kafka/consumer/consumer.go (Task 17
// brief). Every player-npcs Kafka consumer resolves its topic and brokers
// through NewConfig so the env-var lookup lives in exactly one place.
package consumer

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

func NewConfig(l logrus.FieldLogger) func(name string) func(token topic.Token) func(groupId string) consumer.Config {
	return func(name string) func(token topic.Token) func(groupId string) consumer.Config {
		return func(token topic.Token) func(groupId string) consumer.Config {
			t, err := topic.EnvProvider(l)(token)()
			if err != nil {
				l.WithError(err).Fatalf("unresolvable topic token [%s]", token)
			}
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
