package consumer

import (
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"os"
)

type Config = consumer.Config

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) Config {
	return func(name string) func(token string) func(groupId string) Config {
		return func(token string) func(groupId string) Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}

func Start(l logrus.FieldLogger, config Config, processor interface{}) {
	// Start the Kafka consumer using the atlas-kafka manager pattern
	l.WithField("config", config).Info("Starting Kafka consumer")
	
	// Get the consumer manager instance  
	manager := consumer.GetManager()
	
	// Add the consumer to the manager
	// This will start the consumer in a background goroutine
	manager.AddConsumer(l, context.Background(), nil)(config)
	
	l.Info("Kafka consumer started successfully")
}
