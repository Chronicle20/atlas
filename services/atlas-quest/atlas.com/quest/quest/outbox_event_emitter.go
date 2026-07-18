package quest

import (
	questmessage "atlas-quest/kafka/message/quest"
	sagamessage "atlas-quest/kafka/message/saga"
	questproducer "atlas-quest/kafka/producer/quest"
	sagaproducer "atlas-quest/kafka/producer/saga"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// OutboxEventEmitter persists events as outbox rows inside tx instead of
// publishing directly; the drainer publishes after commit.
type OutboxEventEmitter struct {
	l   logrus.FieldLogger
	ctx context.Context
	tx  *gorm.DB
}

func NewOutboxEventEmitter(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) EventEmitter {
	return &OutboxEventEmitter{l: l, ctx: ctx, tx: tx}
}

func (e *OutboxEventEmitter) enqueue(token string, p model.Provider[[]kafka.Message]) error {
	msgs, err := p()
	if err != nil {
		return err
	}
	return outbox.EnqueueBuffer(e.l, e.ctx, e.tx, map[string][]kafka.Message{token: msgs})
}

func (e *OutboxEventEmitter) EmitQuestStarted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, progress string, items []questmessage.ItemReward) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestStartedEventProvider(transactionId, characterId, worldId, questId, progress, items))
}

func (e *OutboxEventEmitter) EmitQuestCompleted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, completedAt time.Time, items []questmessage.ItemReward) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestCompletedEventProvider(transactionId, characterId, worldId, questId, completedAt, items))
}

func (e *OutboxEventEmitter) EmitQuestForfeited(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestForfeitedEventProvider(transactionId, characterId, worldId, questId))
}

func (e *OutboxEventEmitter) EmitProgressUpdated(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, infoNumber uint32, progress string) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestProgressUpdatedEventProvider(transactionId, characterId, worldId, questId, infoNumber, progress))
}

func (e *OutboxEventEmitter) EmitSaga(s sagamessage.Saga) error {
	return e.enqueue(sagamessage.EnvCommandTopic, sagaproducer.SagaCommandProvider(s))
}
