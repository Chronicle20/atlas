// Package trade consumes COMMAND_TOPIC_TRADE, the atlas-channel -> atlas-trades
// command stream. The topic is SHARED by every handler registered on it, so each
// handler discriminates on Command.Type BEFORE it uses Body — a handler that
// skipped the guard would unmarshal another command's body into its own shape
// and act on a zero value (see trademsg.Command's doc comment).
package trade

import (
	consumer2 "atlas-trades/kafka/consumer"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/trade"
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
			rf(consumer2.NewConfig(l)("trade_command")(trademsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(trademsg.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateRoom(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleInvite(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeclineInvite(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterRoom(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// fieldFromCommand rebuilds the acting character's field from the envelope.
func fieldFromCommand[E any](c trademsg.Command[E]) field.Model {
	return field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
}

func handleCreateRoom(db *gorm.DB) message.Handler[trademsg.Command[trademsg.CreateRoomCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.CreateRoomCommandBody]) {
		if c.Type != trademsg.CommandTypeCreateRoom {
			return
		}
		f := fieldFromCommand(c)
		if err := trade.NewProcessor(l, ctx, db).CreateRoom(c.TransactionId, f, c.CharacterId, c.Body.RoomType); err != nil {
			l.WithError(err).Errorf("Unable to open a trade room for character [%d].", c.CharacterId)
		}
	}
}

func handleInvite(db *gorm.DB) message.Handler[trademsg.Command[trademsg.InviteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.InviteCommandBody]) {
		if c.Type != trademsg.CommandTypeInvite {
			return
		}
		f := fieldFromCommand(c)
		if err := trade.NewProcessor(l, ctx, db).Invite(c.TransactionId, f, c.CharacterId, c.Body.TargetCharacterId); err != nil {
			l.WithError(err).Errorf("Unable to invite character [%d] to character [%d]'s trade.", c.Body.TargetCharacterId, c.CharacterId)
		}
	}
}

// handleDeclineInvite resolves the declined room from the wire serial the client
// echoed back. The room's owner — not the serial itself — is what identifies the
// originator: the serial happens to default to the owner's character id
// (design §2.3), but it is a wire handle and must not be read as a character id.
func handleDeclineInvite(db *gorm.DB) message.Handler[trademsg.Command[trademsg.DeclineInviteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.DeclineInviteCommandBody]) {
		if c.Type != trademsg.CommandTypeDeclineInvite {
			return
		}
		p := trade.NewProcessor(l, ctx, db)
		room, ok := p.RoomByHandle(c.Body.SerialNumber)
		if !ok {
			l.Debugf("Character [%d] declined trade invite [%d], whose room is already gone.", c.CharacterId, c.Body.SerialNumber)
			return
		}
		if err := p.DeclineInvite(c.TransactionId, c.CharacterId, room.OwnerId()); err != nil {
			l.WithError(err).Errorf("Unable to decline character [%d]'s trade invite to [%d].", room.OwnerId(), c.CharacterId)
		}
	}
}

func handleEnterRoom(db *gorm.DB) message.Handler[trademsg.Command[trademsg.EnterRoomCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.EnterRoomCommandBody]) {
		if c.Type != trademsg.CommandTypeEnterRoom {
			return
		}
		f := fieldFromCommand(c)
		if err := trade.NewProcessor(l, ctx, db).EnterRoom(c.TransactionId, f, c.CharacterId, c.Body.Handle); err != nil {
			l.WithError(err).Errorf("Unable to seat character [%d] in trade room [%d].", c.CharacterId, c.Body.Handle)
		}
	}
}
