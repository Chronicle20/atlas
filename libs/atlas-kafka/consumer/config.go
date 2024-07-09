package consumer

import (
	"os"
	"time"
)

func NewEnvConfig(name string, topic string, groupId string) Config {
	brokers := []string{os.Getenv("BOOTSTRAP_SERVERS")}
	return NewConfig(brokers, name, topic, groupId)
}

func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers: brokers,
		name:    name,
		topic:   topic,
		groupId: groupId,
		maxWait: 500 * time.Millisecond,
	}
}

type Config struct {
	brokers []string
	name    string
	topic   string
	groupId string
	maxWait time.Duration
}
