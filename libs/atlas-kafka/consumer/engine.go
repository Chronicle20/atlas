package consumer

import (
	"context"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// EngineName selects the consumer implementation.
type EngineName string

const (
	// EngineConsumerGroup is the supported engine: kafka.ConsumerGroup +
	// Generation, so partition assignment is directly observable and a
	// stalled partition is recovered in place instead of by rejoining the
	// group (task-209).
	EngineConsumerGroup EngineName = "consumergroup"

	// EngineReader is the legacy *kafka.Reader path, retained for one
	// release as the rollback target. Both engines use the same group IDs
	// and commit the same offset value (msg.Offset+1), so switching is a pod
	// restart with no topic, offset or group-state migration (FR-5.2/5.3).
	EngineReader EngineName = "reader"
)

// engineEnvVar selects the engine at process start. Unset or empty means
// EngineConsumerGroup; an unrecognised value warns and falls back to the
// default, because a typo in a deployment env var must not take a service's
// consumers offline.
const engineEnvVar = "KAFKA_CONSUMER_ENGINE"

// ConfigEngine pins the engine explicitly, overriding engineEnvVar. It exists
// so tests can select an engine without mutating process env, and so an
// embedder can hard-select one.
//
//goland:noinspection GoUnusedExportedFunction
func ConfigEngine(e EngineName) ManagerConfig {
	return func(m *Manager) {
		m.engine = e
	}
}

func resolveEngine(l logrus.FieldLogger) EngineName {
	v := os.Getenv(engineEnvVar)
	switch EngineName(v) {
	case "":
		return EngineConsumerGroup
	case EngineConsumerGroup:
		return EngineConsumerGroup
	case EngineReader:
		return EngineReader
	default:
		l.Warnf("Unrecognised %s value [%s]; using [%s].", engineEnvVar, v, EngineConsumerGroup)
		return EngineConsumerGroup
	}
}

// start dispatches to the configured engine. It is the single entry point
// AddConsumer launches per consumer.
func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	if c.engine == EngineReader {
		c.startReaderEngine(l, ctx, wg)
		return
	}
	c.startGroupEngine(l, ctx, wg)
}
