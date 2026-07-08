package saga

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// createCommandProvider builds the single-message Kafka provider for
// submitting s to atlas-saga-orchestrator's command topic, keyed by the
// saga's transaction id. Mirrors
// atlas-npc-conversations/atlas.com/npc/saga/producer.go.
func createCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
