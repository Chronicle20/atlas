package consumer

import (
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// NewConfig builds a consumer Config with the library defaults.
//
// Default rationale (task-136 — see docs/tasks/task-136-consumer-fetch-wedge/findings.md):
//
//   - maxWait 10s: kafka-go's own default. With MinBytes=1 (the kafka-go
//     default) the broker answers a fetch immediately when data exists;
//     MaxWait only bounds how long the broker parks an EMPTY long-poll, so
//     a large value costs zero delivery latency while cutting idle fetch
//     traffic ~200× vs the previous 50ms.
//   - fetchTimeout 1m: the per-call FetchMessage deadline is a liveness
//     tick, not a recreate trigger. A deadline expiration on a reader that
//     is still making fetch attempts is an idle tick (healthy); only ticks
//     with zero reader progress count toward maxConsecutiveTimeouts.
//   - maxConsecutiveTimeouts 3: consecutive NO-PROGRESS ticks before the
//     reader is declared wedged and recreated (~3m to detection at the
//     default tick interval).
//
//goland:noinspection GoUnusedExportedFunction
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers:                brokers,
		name:                   name,
		topic:                  topic,
		groupId:                groupId,
		maxWait:                10 * time.Second,
		startOffset:            kafka.FirstOffset,
		fetchTimeout:           time.Minute,
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

// SetMaxWait should stay comfortably below fetchTimeout: an idle reader's
// Stats().Fetches increments about once per maxWait interval, so
// fetchTimeout needs at least one such interval to observe a fetch attempt
// and classify the tick as idle rather than no-progress.
//
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

// SetFetchTimeout should be set comfortably above maxWait: this is the
// per-call FetchMessage liveness-tick deadline, and an idle reader only
// completes a fetch attempt (registered via Stats().Fetches) about once per
// maxWait, so fetchTimeout <= maxWait risks misclassifying a healthy idle
// reader as no-progress and recreating it needlessly.
//
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
