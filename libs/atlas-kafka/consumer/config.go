package consumer

import (
	"github.com/Chronicle20/atlas-model/model"
	"time"
)

//goland:noinspection GoUnusedExportedFunction
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers: brokers,
		name:    name,
		topic:   topic,
		groupId: groupId,
		maxWait: 50 * time.Millisecond,
	}
}

type Config struct {
	brokers       []string
	name          string
	topic         string
	groupId       string
	maxWait       time.Duration
	headerParsers []HeaderParser
}

//goland:noinspection GoUnusedExportedFunction
func SetMaxWait(duration time.Duration) model.Decorator[Config] {
	return func(config Config) Config {
		config.maxWait = duration
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetHeaderParsers(parsers ...HeaderParser) model.Decorator[Config] {
	return func(config Config) Config {
		config.headerParsers = parsers
		return config
	}
}
