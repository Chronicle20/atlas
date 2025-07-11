package family

import (
	"context"
	"errors"
	
	"atlas-family/family"
	"atlas-family/kafka/consumer"
	"atlas-family/kafka/message"
	familymsg "atlas-family/kafka/message/family"
	"github.com/Chronicle20/atlas-kafka/consumer/kafka"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// FamilyConsumer handles family-related Kafka commands
type FamilyConsumer struct {
	db        *gorm.DB
	log       logrus.FieldLogger
	processor family.Processor
	admin     family.Administrator
}

// NewFamilyConsumer creates a new family consumer instance
func NewFamilyConsumer(db *gorm.DB, log logrus.FieldLogger, processor family.Processor, admin family.Administrator) *FamilyConsumer {
	return &FamilyConsumer{
		db:        db,
		log:       log,
		processor: processor,
		admin:     admin,
	}
}

// Config returns the consumer configuration for family commands
func (fc *FamilyConsumer) Config(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) kafka.Config {
	return consumer.NewConfig(l)
}

// InitConsumers initializes the family command consumers
func InitConsumers(l logrus.FieldLogger, db *gorm.DB, processor family.Processor, admin family.Administrator) func(func(config kafka.Config, decorators ...model.Decorator[kafka.Config])) func(consumerGroupId string) {
	return func(rf func(config kafka.Config, decorators ...model.Decorator[kafka.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			consumer := NewFamilyConsumer(db, l, processor, admin)
			
			// Configure consumer for family commands
			rf(consumer.Config(l)("family_commands")(familymsg.EnvCommandTopic)(consumerGroupId),
				kafka.SetHeaderParsers(kafka.SpanHeaderParser, kafka.TenantHeaderParser))
		}
	}
}

// InitHandlers initializes the Kafka message handlers
func InitHandlers(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, processor family.Processor, admin family.Administrator) {
	consumer := NewFamilyConsumer(db, l, processor, admin)
	
	// Register handlers for different command types
	message.AdaptHandler(l, ctx, handleAddJuniorCommand(consumer))
	message.AdaptHandler(l, ctx, handleRemoveMemberCommand(consumer))
	message.AdaptHandler(l, ctx, handleBreakLinkCommand(consumer))
	message.AdaptHandler(l, ctx, handleDeductRepCommand(consumer))
	message.AdaptHandler(l, ctx, handleRegisterKillActivityCommand(consumer))
	message.AdaptHandler(l, ctx, handleRegisterExpeditionActivityCommand(consumer))
}

// handleAddJuniorCommand handles add junior commands
func handleAddJuniorCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.AddJuniorCommandBody]) {
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
		
		// Parse tenant ID
		tenantId, err := uuid.Parse(cmd.TenantId)
		if err != nil {
			l.WithError(err).Error("Failed to parse tenant ID")
			return
		}
		
		// Ensure both senior and junior exist as family members
		_, err = consumer.admin.EnsureMemberExists(consumer.db, l)(cmd.CharacterId, tenantId, cmd.Body.SeniorLevel, cmd.Body.SeniorWorld)()
		if err != nil {
			l.WithError(err).Error("Failed to ensure senior member exists")
			return
		}
		
		_, err = consumer.admin.EnsureMemberExists(consumer.db, l)(cmd.Body.JuniorId, tenantId, cmd.Body.JuniorLevel, cmd.Body.JuniorWorld)()
		if err != nil {
			l.WithError(err).Error("Failed to ensure junior member exists")
			return
		}
		
		// Process the add junior operation
		_, err = consumer.processor.AddJuniorAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.JuniorId)()
		if err != nil {
			l.WithError(err).Error("Failed to process add junior command")
			return
		}
		
		l.Info("Successfully processed add junior command")
	}
}

// handleRemoveMemberCommand handles remove member commands
func handleRemoveMemberCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.RemoveMemberCommandBody]) {
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
		_, err := consumer.processor.RemoveMemberAndEmit(cmd.TransactionId, cmd.Body.TargetId, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process remove member command")
			return
		}
		
		l.Info("Successfully processed remove member command")
	}
}

// handleBreakLinkCommand handles break link commands
func handleBreakLinkCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.BreakLinkCommandBody]) {
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
		_, err := consumer.processor.BreakLinkAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process break link command")
			return
		}
		
		l.Info("Successfully processed break link command")
	}
}

// handleDeductRepCommand handles deduct reputation commands
func handleDeductRepCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.DeductRepCommandBody]) {
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
		_, err := consumer.processor.DeductRepAndEmit(cmd.TransactionId, cmd.CharacterId, cmd.Body.Amount, cmd.Body.Reason)()
		if err != nil {
			l.WithError(err).Error("Failed to process deduct reputation command")
			return
		}
		
		l.Info("Successfully processed deduct reputation command")
	}
}

// handleRegisterKillActivityCommand handles register kill activity commands
func handleRegisterKillActivityCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.RegisterKillActivityCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.RegisterKillActivityCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"killCount":     cmd.Body.KillCount,
			"type":          cmd.Type,
		}).Info("Processing register kill activity command")
		
		// Validate command type
		if cmd.Type != familymsg.CommandTypeRegisterKillActivity {
			l.WithField("type", cmd.Type).Warn("Ignoring non-register-kill-activity command")
			return
		}
		
		// Process the kill activity and award reputation to senior if applicable
		_, err := consumer.admin.ProcessRepActivity(consumer.db, l)(cmd.CharacterId, "mob_kill", cmd.Body.KillCount)()
		if err != nil {
			if errors.Is(err, family.ErrMemberNotFound) {
				l.WithField("characterId", cmd.CharacterId).Debug("Character not found, no reputation to award")
				return
			}
			l.WithError(err).Error("Failed to process kill activity")
			return
		}
		
		l.Info("Successfully processed register kill activity command")
	}
}

// handleRegisterExpeditionActivityCommand handles register expedition activity commands
func handleRegisterExpeditionActivityCommand(consumer *FamilyConsumer) func(logrus.FieldLogger, context.Context, familymsg.Command[familymsg.RegisterExpeditionActivityCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd familymsg.Command[familymsg.RegisterExpeditionActivityCommandBody]) {
		l.WithFields(logrus.Fields{
			"transactionId": cmd.TransactionId,
			"characterId":   cmd.CharacterId,
			"coinReward":    cmd.Body.CoinReward,
			"type":          cmd.Type,
		}).Info("Processing register expedition activity command")
		
		// Validate command type
		if cmd.Type != familymsg.CommandTypeRegisterExpeditionActivity {
			l.WithField("type", cmd.Type).Warn("Ignoring non-register-expedition-activity command")
			return
		}
		
		// Process the expedition activity and award reputation to senior if applicable
		_, err := consumer.admin.ProcessRepActivity(consumer.db, l)(cmd.CharacterId, "expedition", cmd.Body.CoinReward)()
		if err != nil {
			if errors.Is(err, family.ErrMemberNotFound) {
				l.WithField("characterId", cmd.CharacterId).Debug("Character not found, no reputation to award")
				return
			}
			l.WithError(err).Error("Failed to process expedition activity")
			return
		}
		
		l.Info("Successfully processed register expedition activity command")
	}
}