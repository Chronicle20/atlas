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
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePutItem(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAddMeso(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleConfirm(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleTransaction(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancel(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChat(db)))); err != nil {
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

// handlePutItem stages one item. The body's InventoryType is the shared
// inventory.Type (a SIGNED int8), so a wire byte above 127 has already arrived
// here negative; it is handed to the processor as the raw byte it was, and the
// processor's decode boundary rejects anything outside the five compartments.
func handlePutItem(db *gorm.DB) message.Handler[trademsg.Command[trademsg.PutItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.PutItemCommandBody]) {
		if c.Type != trademsg.CommandTypePutItem {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).PutItem(c.TransactionId, c.CharacterId, byte(c.Body.InventoryType), c.Body.Slot, c.Body.Quantity, c.Body.TargetSlot); err != nil {
			l.WithError(err).Errorf("Unable to stage an item for character [%d].", c.CharacterId)
		}
	}
}

func handleAddMeso(db *gorm.DB) message.Handler[trademsg.Command[trademsg.AddMesoCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.AddMesoCommandBody]) {
		if c.Type != trademsg.CommandTypeAddMeso {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).AddMeso(c.TransactionId, c.CharacterId, c.Body.Amount); err != nil {
			l.WithError(err).Errorf("Unable to stage meso for character [%d].", c.CharacterId)
		}
	}
}

// handleConfirm records one side pressing Trade. The CRC list rides along and
// is kept as that side's fallback attestation for the timeout path (design
// §3.1); it is empty on the versions whose TRADE_CONFIRM carries none.
func handleConfirm(db *gorm.DB) message.Handler[trademsg.Command[trademsg.ConfirmCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.ConfirmCommandBody]) {
		if c.Type != trademsg.CommandTypeConfirm {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).Confirm(c.TransactionId, c.CharacterId, c.Body.Entries); err != nil {
			l.WithError(err).Errorf("Unable to confirm character [%d]'s trade.", c.CharacterId)
		}
	}
}

// handleTransaction records the client's automatic CRC attestation. TRANSACTION
// is NOT a user action — CTradingRoomDlg::OnTrade sends it on receipt of
// clientbound mode 17 (design §1.5) — so it arrives once per side, unprompted
// by the player.
func handleTransaction(db *gorm.DB) message.Handler[trademsg.Command[trademsg.TransactionCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.TransactionCommandBody]) {
		if c.Type != trademsg.CommandTypeTransaction {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).Attest(c.TransactionId, c.CharacterId, c.Body.Entries); err != nil {
			l.WithError(err).Errorf("Unable to record character [%d]'s trade attestation.", c.CharacterId)
		}
	}
}

func handleEnterRoom(db *gorm.DB) message.Handler[trademsg.Command[trademsg.EnterRoomCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.EnterRoomCommandBody]) {
		if c.Type != trademsg.CommandTypeEnterRoom {
			return
		}
		f := fieldFromCommand(c)
		if err := trade.NewProcessor(l, ctx, db).EnterRoom(c.TransactionId, f, c.CharacterId, c.Body.Handle, c.Body.RoomType); err != nil {
			l.WithError(err).Errorf("Unable to seat character [%d] in trade room [%d].", c.CharacterId, c.Body.Handle)
		}
	}
}

// handleCancel is the client closing its trade dialog — the serverbound EXIT
// mode, which atlas-channel fans out to every mini-room family. It is the ONLY
// teardown trigger that fires while the player stays logged in on the same map:
// the character and session consumers cover LOGOUT, MAP_CHANGED,
// CHANNEL_CHANGED and SESSION_DESTROYED, none of which a dialog close produces.
// TeardownCharacter is a no-op for a character with no room, so the
// unconditional fan-out costs nothing.
func handleCancel(db *gorm.DB) message.Handler[trademsg.Command[trademsg.CancelCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.CancelCommandBody]) {
		if c.Type != trademsg.CommandTypeCancel {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).TeardownCharacter(c.TransactionId, c.CharacterId, trade.ReasonTradeCancelled); err != nil {
			l.WithError(err).Errorf("Unable to cancel character [%d]'s trade.", c.CharacterId)
		}
	}
}

// handleChat relays a trade-room chat line. Like CANCEL it arrives for every
// mini-room family, and the processor drops a speaker who is not in a room.
func handleChat(db *gorm.DB) message.Handler[trademsg.Command[trademsg.ChatCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c trademsg.Command[trademsg.ChatCommandBody]) {
		if c.Type != trademsg.CommandTypeChat {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).Chat(c.TransactionId, c.CharacterId, c.Body.Message); err != nil {
			l.WithError(err).Errorf("Unable to relay character [%d]'s trade chat.", c.CharacterId)
		}
	}
}
