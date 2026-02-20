package family

import (
	"atlas-family/family"
	consumer2 "atlas-family/kafka/consumer"
	familymsg "atlas-family/kafka/message/family"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("family_command")(familymsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(familymsg.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAddJuniorCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveMemberCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBreakLinkCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardRepCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeductRepCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleAddJuniorCommand handles add junior commands
func handleAddJuniorCommand(db *gorm.DB) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.AddJuniorCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.AddJuniorCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"juniorId":      cmd.Body.JuniorId,
			"type":          cmd.Type,
		}).Info("Processing add junior command")

		// Validate command type
		if cmd.Type != familymsg.CommandTypeAddJunior {
			l.WithField("type", cmd.Type).Warn("Ignoring non-add-junior command")
			return
		}

		fp := family.NewProcessor(l, ctx, db)

		// Ensure both senior and junior exist as family members
		_, err := fp.GetByCharacterId(cmd.CharacterId)
		if err != nil {
			l.WithError(err).Error("Failed to ensure senior member exists")
			return
		}

		_, err = fp.GetByCharacterId(cmd.Body.JuniorId)
		if err != nil {
			l.WithError(err).Error("Failed to ensure junior member exists")
			return
		}

		// Process the add junior operation
		_, err = fp.AddJuniorAndEmit(cmd.TransactionId, cmd.WorldId, cmd.CharacterId, cmd.Body.SeniorLevel, cmd.Body.JuniorId, cmd.Body.JuniorLevel)()
		if err != nil {
			l.WithError(err).Error("Failed to process add junior command")
			return
		}

		l.Info("Successfully processed add junior command")
	}
}

// handleRemoveMemberCommand handles remove member commands
func handleRemoveMemberCommand(db *gorm.DB) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.RemoveMemberCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.RemoveMemberCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"targetId":      cmd.Body.TargetId,
			"reason":        cmd.Body.Reason,
			"type":          cmd.Type,
		}).Info("Processing remove member command")

		// Validate command type
		if cmd.Type != familymsg.CommandTypeRemoveMember {
			l.WithField("type", cmd.Type).Warn("Ignoring non-remove-member command")
			return
		}

		// Process the remove member operation
		_, err := family.NewProcessor(l, ctx, db).RemoveMemberAndEmit(cmd.TransactionId, cmd.Body.TargetId, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process remove member command")
			return
		}

		l.Info("Successfully processed remove member command")
	}
}

// handleBreakLinkCommand handles break link commands
func handleBreakLinkCommand(db *gorm.DB) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.BreakLinkCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.BreakLinkCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"reason":        cmd.Body.Reason,
			"type":          cmd.Type,
		}).Info("Processing break link command")

		// Validate command type
		if cmd.Type != familymsg.CommandTypeBreakLink {
			l.WithField("type", cmd.Type).Warn("Ignoring non-break-link command")
			return
		}

		// Process the break link operation
		_, err := family.NewProcessor(l, ctx, db).BreakLinkAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process break link command")
			return
		}

		l.Info("Successfully processed break link command")
	}
}

// handleAwardRepCommand handles award reputation commands
func handleAwardRepCommand(db *gorm.DB) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.AwardRepCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.AwardRepCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"amount":        cmd.Body.Amount,
			"source":        cmd.Body.Source,
			"type":          cmd.Type,
		}).Info("Processing award reputation command")

		// Validate command type
		if cmd.Type != familymsg.CommandTypeAwardRep {
			l.WithField("type", cmd.Type).Warn("Ignoring non-award-rep command")
			return
		}

		// Process the deduct reputation operation
		_, err := family.NewProcessor(l, ctx, db).AwardRepAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.Amount, cmd.Body.Source)()
		if err != nil {
			l.WithError(err).Error("Failed to process award reputation command")
			return
		}

		l.Info("Successfully processed award reputation command")
	}
}

// handleDeductRepCommand handles deduct reputation commands
func handleDeductRepCommand(db *gorm.DB) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.DeductRepCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.DeductRepCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"amount":        cmd.Body.Amount,
			"reason":        cmd.Body.Reason,
			"type":          cmd.Type,
		}).Info("Processing deduct reputation command")

		// Validate command type
		if cmd.Type != familymsg.CommandTypeDeductRep {
			l.WithField("type", cmd.Type).Warn("Ignoring non-deduct-rep command")
			return
		}

		// Process the deduct reputation operation
		_, err := family.NewProcessor(l, ctx, db).DeductRepAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.Amount, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process deduct reputation command")
			return
		}

		l.Info("Successfully processed deduct reputation command")
	}
}
