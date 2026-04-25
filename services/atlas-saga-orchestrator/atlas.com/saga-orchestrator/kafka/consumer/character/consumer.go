package character

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterMapChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterExperienceChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterLevelChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterMesoChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterJobChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterCreationFailedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterMesoErrorEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterStatChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterDeletedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleCharacterMapChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type != character2.StatusEventTypeMapChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMapChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCharacterExperienceChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ExperienceChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeExperienceChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterExperienceChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCharacterLevelChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LevelChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeLevelChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterLevelChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCharacterMesoChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.MesoChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeMesoChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMesoChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCharacterJobChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.JobChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeJobChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterJobChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCharacterCreatedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventCreatedBody]) {
	if e.Type != character2.StatusEventTypeCreated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterCreated); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
		"character_name": e.Body.Name,
		"world_id":       e.WorldId,
	}).Debug("Character created successfully, marking saga step as completed")

	_ = p.StepCompletedWithResult(e.TransactionId, true, map[string]any{"characterId": e.CharacterId})
}

func handleCharacterCreationFailedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventCreationFailedBody]) {
	if e.Type != character2.StatusEventTypeCreationFailed {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterCreationFailed); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_name": e.Body.Name,
		"error_message":  e.Body.Message,
		"world_id":       e.WorldId,
	}).Error("Character creation failed, marking saga step as failed")

	_ = p.StepCompleted(e.TransactionId, false)
}

func handleCharacterMesoErrorEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMesoErrorBody]) {
	if e.Type != character2.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMesoError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
		"error_type":     e.Body.Error,
		"amount":         e.Body.Amount,
		"world_id":       e.WorldId,
	}).Error("Character meso operation error occurred, marking saga step as failed")

	_ = p.StepCompleted(e.TransactionId, false)
}

func handleCharacterStatChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
	if e.Type != character2.StatusEventTypeStatChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterStatChanged); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

// handleCharacterDeletedEvent drives StepCompleted(true) when atlas-character
// responds to a saga-correlated DELETE_CHARACTER command (the reverse-walk
// compensator; plan Phase 5 / Phase 6).
func handleCharacterDeletedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventDeletedBody]) {
	if e.Type != character2.StatusEventTypeDeleted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterDeleted); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
	}).Debug("Character deleted, marking saga compensation step completed.")
	_ = p.StepCompleted(e.TransactionId, true)
}
