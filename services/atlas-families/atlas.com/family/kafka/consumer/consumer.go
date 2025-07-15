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
	// This is a placeholder for the actual consumer start logic
	// In a real implementation, this would start the Kafka consumer
	// and process messages using the provided processor
	l.WithField("config", config).Info("Starting Kafka consumer")
	
	// Create a consumer instance and start consuming
	c := consumer.New(config)
	
	// Start the consumer in a background goroutine
	go func() {
		if err := c.Start(context.Background()); err != nil {
			l.WithError(err).Error("Failed to start Kafka consumer")
		}
	}()
}
