package cashshop

import (
	"atlas-maps/character/location"
	consumer2 "atlas-maps/kafka/consumer"
	cashshopKafka "atlas-maps/kafka/message/cashshop"
	_map "atlas-maps/map"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(cashshopKafka.EnvEventTopicCashShopStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(cashshopKafka.EnvEventTopicCashShopStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventEnterFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExitFunc(db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventEnterFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
		if event.Type != cashshopKafka.EventCashShopStatusTypeCharacterEnter {
			return
		}
		l.Debugf("Character [%d] has entered cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)
		_ = p.ExitAndEmit(transactionId, f, event.Body.CharacterId)

		// Conditional: OFFLINE is terminal except via LOGIN / CHANNEL_CHANGED.
		// The cash-shop and character status topics have no mutual ordering
		// guarantee, so a late ENTER must not resurrect a logged-off row.
		if err := location.NewProcessor(l, ctx, db).SetStateIfOnline(event.Body.CharacterId, characterconst.PresenceStateInCashShop); err != nil {
			l.WithError(err).Warnf("location.SetStateIfOnline on CHARACTER_ENTER failed for character [%d].", event.Body.CharacterId)
		}
	}
}

func handleStatusEventExitFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
		if event.Type != cashshopKafka.EventCashShopStatusTypeCharacterExit {
			return
		}
		l.Debugf("Character [%d] has exited cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)
		_ = p.EnterAndEmit(transactionId, f, event.Body.CharacterId)

		// Conditional for the same reason: disconnecting from inside the cash
		// shop emits LOGOUT and no CHARACTER_EXIT, so an EXIT that arrives
		// after that LOGOUT must leave the row OFFLINE.
		if err := location.NewProcessor(l, ctx, db).SetStateIfOnline(event.Body.CharacterId, characterconst.PresenceStateInField); err != nil {
			l.WithError(err).Warnf("location.SetStateIfOnline on CHARACTER_EXIT failed for character [%d].", event.Body.CharacterId)
		}
	}
}
