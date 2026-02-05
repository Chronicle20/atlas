package quest

import (
	questmessage "atlas-quest/kafka/message/quest"
	"atlas-quest/kafka/message/saga"
	questproducer "atlas-quest/kafka/producer/quest"
	sagaproducer "atlas-quest/kafka/producer/saga"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EventEmitter defines the interface for emitting quest-related events
type EventEmitter interface {
	EmitQuestStarted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, progress string) error
	EmitQuestCompleted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, completedAt time.Time, items []questmessage.ItemReward) error
	EmitQuestForfeited(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32) error
	EmitProgressUpdated(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, infoNumber uint32, progress string) error
	EmitSaga(s saga.Saga) error
}

// KafkaEventEmitter is the production implementation that emits to Kafka
type KafkaEventEmitter struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewKafkaEventEmitter creates a new Kafka-based event emitter
func NewKafkaEventEmitter(l logrus.FieldLogger, ctx context.Context) EventEmitter {
	return &KafkaEventEmitter{l: l, ctx: ctx}
}

func (e *KafkaEventEmitter) EmitQuestStarted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, progress string) error {
	return questproducer.EmitQuestStarted(e.l, e.ctx, transactionId, characterId, worldId, questId, progress)
}

func (e *KafkaEventEmitter) EmitQuestCompleted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, completedAt time.Time, items []questmessage.ItemReward) error {
	return questproducer.EmitQuestCompleted(e.l, e.ctx, transactionId, characterId, worldId, questId, completedAt, items)
}

func (e *KafkaEventEmitter) EmitQuestForfeited(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32) error {
	return questproducer.EmitQuestForfeited(e.l, e.ctx, transactionId, characterId, worldId, questId)
}

func (e *KafkaEventEmitter) EmitProgressUpdated(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, infoNumber uint32, progress string) error {
	return questproducer.EmitProgressUpdated(e.l, e.ctx, transactionId, characterId, worldId, questId, infoNumber, progress)
}

func (e *KafkaEventEmitter) EmitSaga(s saga.Saga) error {
	return sagaproducer.EmitSaga(e.l, e.ctx, s)
}
