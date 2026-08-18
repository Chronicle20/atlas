// Package custody consumes COMMAND_TOPIC_TRADE_CUSTODY, the
// atlas-saga-orchestrator -> atlas-trades escrow custody stream (task-205
// design §5A.2).
//
// Like every other command topic in this service it is SHARED by all handlers
// registered on it, so each handler discriminates on Command.Type BEFORE it
// touches Body — a handler that skipped the guard would unmarshal another
// command's body into its own shape and act on a zero value. For this topic
// that would mean writing or deleting an escrow row keyed by the nil UUID.
package custody

import (
	"atlas-trades/escrow"
	consumer2 "atlas-trades/kafka/consumer"
	custodymsg "atlas-trades/kafka/message/custody"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_custody_command")(custodymsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(custodymsg.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAccept(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRelease(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRestore(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemove(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleAccept writes the escrow row for an item that has already left its
// owner's compartment. By the time this runs the asset is gone from
// atlas-inventory, so a failure here is not cosmetic — the ack path turns it
// into a saga failure, whose reverse-walk re-grants the item.
func handleAccept(db *gorm.DB) message.Handler[custodymsg.Command[custodymsg.AcceptToTradeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c custodymsg.Command[custodymsg.AcceptToTradeCommandBody]) {
		if c.Type != custodymsg.CommandAcceptToTrade {
			return
		}
		b := c.Body
		m := escrow.NewItemBuilder(b.EscrowId, b.RoomId, character.Id(b.OwnerId)).
			SetTradeSlot(b.TradeSlot).
			SetSource(inventory.Type(b.SourceInventoryType), asset.Id(b.AssetId)).
			SetSnapshot(b.Snapshot).
			Build()

		if err := escrow.NewProcessor(l, ctx, db).Accept(c.TransactionId, m); err != nil {
			l.WithError(err).Errorf("Unable to accept trade escrow row [%s] for character [%d].", b.EscrowId, b.OwnerId)
		}
	}
}

func handleRelease(db *gorm.DB) message.Handler[custodymsg.Command[custodymsg.ReleaseFromTradeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c custodymsg.Command[custodymsg.ReleaseFromTradeCommandBody]) {
		if c.Type != custodymsg.CommandReleaseFromTrade {
			return
		}
		if err := escrow.NewProcessor(l, ctx, db).Release(c.TransactionId, c.Body.EscrowId); err != nil {
			l.WithError(err).Errorf("Unable to release trade escrow row [%s].", c.Body.EscrowId)
		}
	}
}

func handleRestore(db *gorm.DB) message.Handler[custodymsg.Command[custodymsg.RestoreTradeEscrowCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c custodymsg.Command[custodymsg.RestoreTradeEscrowCommandBody]) {
		if c.Type != custodymsg.CommandRestoreTradeEscrow {
			return
		}
		if err := escrow.NewProcessor(l, ctx, db).Restore(c.TransactionId, c.Body.EscrowId); err != nil {
			l.WithError(err).Errorf("Unable to restore trade escrow row [%s].", c.Body.EscrowId)
		}
	}
}

func handleRemove(db *gorm.DB) message.Handler[custodymsg.Command[custodymsg.RemoveTradeEscrowCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c custodymsg.Command[custodymsg.RemoveTradeEscrowCommandBody]) {
		if c.Type != custodymsg.CommandRemoveTradeEscrow {
			return
		}
		if err := escrow.NewProcessor(l, ctx, db).Remove(c.TransactionId, c.Body.EscrowId); err != nil {
			l.WithError(err).Errorf("Unable to remove trade escrow row [%s].", c.Body.EscrowId)
		}
	}
}
