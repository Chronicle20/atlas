package minigame

import (
	"atlas-mini-games/game"
	consumer2 "atlas-mini-games/kafka/consumer"
	"atlas-mini-games/kafka/message/minigame"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mini_game_command")(minigame.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(minigame.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreate(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleVisit(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLeave(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChat(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExpel(db)))); err != nil {
				return err
			}
			// Gameplay command handlers (READY/UNREADY/START/MOVE_STONE/FLIP_CARD/
			// tie/retreat/SKIP/EXIT_AFTER_GAME) are registered in Task 15.
			return nil
		}
	}
}

func fieldFromCommand[E any](c minigame.Command[E]) field.Model {
	return field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
}

func handleCreate(db *gorm.DB) message.Handler[minigame.Command[minigame.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.CreateCommandBody]) {
		if c.Type != minigame.CommandTypeCreate {
			return
		}
		f := fieldFromCommand(c)
		_ = game.NewProcessor(l, ctx, db).Create(c.TransactionId, f, c.CharacterId, c.Body.RoomType, c.Body.Title, c.Body.Private, c.Body.Password, c.Body.PieceType)
	}
}

func handleVisit(db *gorm.DB) message.Handler[minigame.Command[minigame.VisitCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.VisitCommandBody]) {
		if c.Type != minigame.CommandTypeVisit {
			return
		}
		f := fieldFromCommand(c)
		_ = game.NewProcessor(l, ctx, db).Visit(c.TransactionId, f, c.CharacterId, c.Body.RoomId, c.Body.Password)
	}
}

func handleLeave(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeLeave {
			return
		}
		f := fieldFromCommand(c)
		_ = game.NewProcessor(l, ctx, db).Leave(c.TransactionId, f, c.CharacterId)
	}
}

func handleChat(db *gorm.DB) message.Handler[minigame.Command[minigame.ChatCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.ChatCommandBody]) {
		if c.Type != minigame.CommandTypeChat {
			return
		}
		f := fieldFromCommand(c)
		_ = game.NewProcessor(l, ctx, db).Chat(c.TransactionId, f, c.CharacterId, c.Body.Message)
	}
}

func handleExpel(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeExpel {
			return
		}
		f := fieldFromCommand(c)
		_ = game.NewProcessor(l, ctx, db).Expel(c.TransactionId, f, c.CharacterId)
	}
}
