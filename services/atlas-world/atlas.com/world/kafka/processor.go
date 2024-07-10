package kafka

import (
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/sirupsen/logrus"
	"os"
)

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t := LookupTopic(l)(token)
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

func LookupTopic(l logrus.FieldLogger) func(token string) string {
	return func(token string) string {
		t, ok := os.LookupEnv(token)
		if !ok {
			l.Warnf("%s environment variable not set. Defaulting to env variable.", token)
			return token

		}
		return t
	}
}

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
