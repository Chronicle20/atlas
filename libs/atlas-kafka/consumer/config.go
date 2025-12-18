package consumer

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

//goland:noinspection GoUnusedExportedFunction
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers:     brokers,
		name:        name,
		topic:       topic,
		groupId:     groupId,
		maxWait:     50 * time.Millisecond,
		startOffset: kafka.FirstOffset,
	}
}

type Config struct {
	brokers       []string
	name          string
	topic         string
	groupId       string
	maxWait       time.Duration
	headerParsers []HeaderParser
	startOffset   int64
}

//goland:noinspection GoUnusedExportedFunction
func SetStartOffset(startOffset int64) model.Decorator[Config] {
	return func(config Config) Config {
		config.startOffset = startOffset
		return config
	}
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
