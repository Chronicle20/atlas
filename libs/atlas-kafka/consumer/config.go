package consumer

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

//goland:noinspection GoUnusedExportedFunction
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers:                brokers,
		name:                   name,
		topic:                  topic,
		groupId:                groupId,
		maxWait:                50 * time.Millisecond,
		startOffset:            kafka.FirstOffset,
		fetchTimeout:           5 * time.Minute,
		maxConsecutiveTimeouts: 3,
	}
}

type Config struct {
	brokers                []string
	name                   string
	topic                  string
	groupId                string
	maxWait                time.Duration
	headerParsers          []HeaderParser
	startOffset            int64
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int
	maxInFlight            int
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

//goland:noinspection GoUnusedExportedFunction
func SetFetchTimeout(d time.Duration) model.Decorator[Config] {
	return func(config Config) Config {
		config.fetchTimeout = d
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetMaxConsecutiveTimeouts(n int) model.Decorator[Config] {
	return func(config Config) Config {
		config.maxConsecutiveTimeouts = n
		return config
	}
}

// SetMaxInFlight enables within-pod parallelism for this consumer. The fetch
// loop spawns up to n handler goroutines concurrently and commits offsets
// using a prefix-commit cursor: only the highest *contiguously*-completed
// message offset is committed, so a failed message in the middle blocks
// subsequent commits (matching today's at-least-once semantics).
//
// Default is 1 (serial — today's behavior). Set to runtime.NumCPU() or a
// service-specific value for high-throughput consumers where messages are
// independent and handlers are CPU-bound.
//
// IMPORTANT: only set this for consumers whose handler is safe for concurrent
// invocation across messages on the same partition. If the handler relies on
// strict in-partition ordering of side effects, leave the default (1).
//
//goland:noinspection GoUnusedExportedFunction
func SetMaxInFlight(n int) model.Decorator[Config] {
	return func(config Config) Config {
		if n < 1 {
			n = 1
		}
		config.maxInFlight = n
		return config
	}
}
