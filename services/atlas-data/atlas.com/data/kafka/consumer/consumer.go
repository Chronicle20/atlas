package consumer

import (
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"os"
)

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

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
