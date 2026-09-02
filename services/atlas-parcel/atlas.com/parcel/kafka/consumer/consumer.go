package consumer

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// NewConfig builds a consumer.Config for a named topic env var and consumer
// group, resolving the topic name and brokers from the environment. Mirrors
// services/atlas-mts/atlas.com/mts/kafka/consumer/consumer.go.
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
