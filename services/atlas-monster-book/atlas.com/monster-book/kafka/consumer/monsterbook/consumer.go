package monsterbook

import (
	"context"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	consumer2 "atlas-monster-book/kafka/consumer"
	"atlas-monster-book/kafka/message"
	mbmsg "atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kmessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_book_command")(mbmsg.EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(mbmsg.EnvCommandTopic)()
			if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleCardPickedUp(db)))); err != nil {
				return err
			}
			if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleSetCover(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleCardPickedUp(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.CardPickedUpBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.CardPickedUpBody]) {
		if cmd.Type != mbmsg.CommandTypeCardPickedUp {
			return
		}
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return message.Emit(producer.ProviderImpl(l)(ctx))(func(mb *message.Buffer) error {
				cp := card.NewProcessor(l, ctx, tx)
				colp := collection.NewProcessor(l, ctx, tx)
				res, err := cp.Add(mb)(cmd.EventId, cmd.CharacterId, cmd.Body.CardId)
				if err != nil {
					return err
				}
				if res.Duplicate {
					return nil
				}
				if res.Inserted {
					return colp.RecomputeAndEmit(mb)(cmd.CharacterId)
				}
				return nil
			})
		})
		if err != nil {
			l.WithError(err).Errorf("Failed to handle CARD_PICKED_UP for character %d card %d.", cmd.CharacterId, cmd.Body.CardId)
		}
	}
}

func handleSetCover(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.SetCoverBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.SetCoverBody]) {
		if cmd.Type != mbmsg.CommandTypeSetCover {
			return
		}
		colp := collection.NewProcessor(l, ctx, db)
		if err := colp.SetCoverAndEmit(cmd.EventId, cmd.CharacterId, cmd.Body.CoverCardId); err != nil {
			l.WithError(err).Warnf("SetCover rejected for character %d cover %d.", cmd.CharacterId, cmd.Body.CoverCardId)
		}
	}
}
