package minigame

import (
	"atlas-mini-games/game"
	consumer2 "atlas-mini-games/kafka/consumer"
	"atlas-mini-games/kafka/message/minigame"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

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
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReady(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUnready(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStart(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveStone(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleFlipCard(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestTie(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAnswerTie(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleGiveUp(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestRetreat(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAnswerRetreat(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkip(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExitAfterGame(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelExitAfterGame(db)))); err != nil {
				return err
			}
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
		if err := game.NewProcessor(l, ctx, db).Create(c.TransactionId, f, c.CharacterId, c.Body.RoomType, c.Body.Title, c.Body.Private, c.Body.Password, c.Body.PieceType); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game create for character [%d].", c.CharacterId)
		}
	}
}

func handleVisit(db *gorm.DB) message.Handler[minigame.Command[minigame.VisitCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.VisitCommandBody]) {
		if c.Type != minigame.CommandTypeVisit {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Visit(c.TransactionId, f, c.CharacterId, c.Body.RoomId, c.Body.Password); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game visit for character [%d].", c.CharacterId)
		}
	}
}

func handleLeave(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeLeave {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Leave(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game leave for character [%d].", c.CharacterId)
		}
	}
}

func handleChat(db *gorm.DB) message.Handler[minigame.Command[minigame.ChatCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.ChatCommandBody]) {
		if c.Type != minigame.CommandTypeChat {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Chat(c.TransactionId, f, c.CharacterId, c.Body.Message); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game chat for character [%d].", c.CharacterId)
		}
	}
}

func handleExpel(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeExpel {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Expel(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game expel for character [%d].", c.CharacterId)
		}
	}
}

func handleReady(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeReady {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Ready(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game ready for character [%d].", c.CharacterId)
		}
	}
}

func handleUnready(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeUnready {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Unready(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game unready for character [%d].", c.CharacterId)
		}
	}
}

func handleStart(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeStart {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Start(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game start for character [%d].", c.CharacterId)
		}
	}
}

func handleMoveStone(db *gorm.DB) message.Handler[minigame.Command[minigame.MoveStoneCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.MoveStoneCommandBody]) {
		if c.Type != minigame.CommandTypeMoveStone {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).MoveStone(c.TransactionId, f, c.CharacterId, c.Body.X, c.Body.Y, c.Body.StoneType); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game move-stone for character [%d].", c.CharacterId)
		}
	}
}

func handleFlipCard(db *gorm.DB) message.Handler[minigame.Command[minigame.FlipCardCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.FlipCardCommandBody]) {
		if c.Type != minigame.CommandTypeFlipCard {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).FlipCard(c.TransactionId, f, c.CharacterId, c.Body.First, c.Body.CardIndex); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game flip-card for character [%d].", c.CharacterId)
		}
	}
}

func handleRequestTie(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeRequestTie {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).RequestTie(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game request-tie for character [%d].", c.CharacterId)
		}
	}
}

func handleAnswerTie(db *gorm.DB) message.Handler[minigame.Command[minigame.AnswerCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.AnswerCommandBody]) {
		if c.Type != minigame.CommandTypeAnswerTie {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).AnswerTie(c.TransactionId, f, c.CharacterId, c.Body.Accept); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game answer-tie for character [%d].", c.CharacterId)
		}
	}
}

func handleGiveUp(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeGiveUp {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).GiveUp(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game give-up for character [%d].", c.CharacterId)
		}
	}
}

func handleRequestRetreat(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeRequestRetreat {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).RequestRetreat(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game request-retreat for character [%d].", c.CharacterId)
		}
	}
}

func handleAnswerRetreat(db *gorm.DB) message.Handler[minigame.Command[minigame.AnswerCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.AnswerCommandBody]) {
		if c.Type != minigame.CommandTypeAnswerRetreat {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).AnswerRetreat(c.TransactionId, f, c.CharacterId, c.Body.Accept); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game answer-retreat for character [%d].", c.CharacterId)
		}
	}
}

func handleSkip(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeSkip {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).Skip(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game skip for character [%d].", c.CharacterId)
		}
	}
}

func handleExitAfterGame(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeExitAfterGame {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).ExitAfterGame(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game exit-after-game for character [%d].", c.CharacterId)
		}
	}
}

func handleCancelExitAfterGame(db *gorm.DB) message.Handler[minigame.Command[minigame.EmptyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c minigame.Command[minigame.EmptyCommandBody]) {
		if c.Type != minigame.CommandTypeCancelExitAfterGame {
			return
		}
		f := fieldFromCommand(c)
		if err := game.NewProcessor(l, ctx, db).CancelExitAfterGame(c.TransactionId, f, c.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to process mini-game cancel-exit-after-game for character [%d].", c.CharacterId)
		}
	}
}
