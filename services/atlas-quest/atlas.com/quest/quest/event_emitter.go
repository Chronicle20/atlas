package quest

import (
	questproducer "atlas-quest/kafka/producer/quest"
	sagaproducer "atlas-quest/kafka/producer/saga"
	"atlas-quest/kafka/message/saga"
	"context"

	"github.com/sirupsen/logrus"
)

// EventEmitter defines the interface for emitting quest-related events
type EventEmitter interface {
	EmitQuestStarted(characterId uint32, worldId byte, questId uint32) error
	EmitQuestCompleted(characterId uint32, worldId byte, questId uint32) error
	EmitQuestForfeited(characterId uint32, worldId byte, questId uint32) error
	EmitProgressUpdated(characterId uint32, worldId byte, questId uint32, infoNumber uint32, progress string) error
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

func (e *KafkaEventEmitter) EmitQuestStarted(characterId uint32, worldId byte, questId uint32) error {
	return questproducer.EmitQuestStarted(e.l, e.ctx, characterId, worldId, questId)
}

func (e *KafkaEventEmitter) EmitQuestCompleted(characterId uint32, worldId byte, questId uint32) error {
	return questproducer.EmitQuestCompleted(e.l, e.ctx, characterId, worldId, questId)
}

func (e *KafkaEventEmitter) EmitQuestForfeited(characterId uint32, worldId byte, questId uint32) error {
	return questproducer.EmitQuestForfeited(e.l, e.ctx, characterId, worldId, questId)
}

func (e *KafkaEventEmitter) EmitProgressUpdated(characterId uint32, worldId byte, questId uint32, infoNumber uint32, progress string) error {
	return questproducer.EmitProgressUpdated(e.l, e.ctx, characterId, worldId, questId, infoNumber, progress)
}

func (e *KafkaEventEmitter) EmitSaga(s saga.Saga) error {
	return sagaproducer.EmitSaga(e.l, e.ctx, s)
}
