package marriage

import (
	"context"

	localConsumer "atlas-marriages/kafka/consumer"
	"atlas-marriages/kafka/message/marriage"
	marriageService "atlas-marriages/marriage"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewConfig creates a new consumer configuration for marriage commands
func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return localConsumer.NewConfig(l)
}

// InitHandlers initializes all marriage command handlers
func InitHandlers(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) []handler.Handler {
	marriageProcessor := marriageService.NewProcessor(l, ctx, db)
	
	return []handler.Handler{
		// Proposal command handlers
		message.AdaptHandler(message.PersistentConfig(handlePropose(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleAccept(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleDecline(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleCancel(l, ctx, marriageProcessor))),
		
		// Ceremony command handlers
		message.AdaptHandler(message.PersistentConfig(handleScheduleCeremony(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleStartCeremony(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleCompleteCeremony(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleCancelCeremony(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handlePostponeCeremony(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleRescheduleCeremony(l, ctx, marriageProcessor))),
		
		// Invitee command handlers
		message.AdaptHandler(message.PersistentConfig(handleAddInvitee(l, ctx, marriageProcessor))),
		message.AdaptHandler(message.PersistentConfig(handleRemoveInvitee(l, ctx, marriageProcessor))),
		
		// Divorce command handler
		message.AdaptHandler(message.PersistentConfig(handleDivorce(l, ctx, marriageProcessor))),
		
		// Advance ceremony state handler
		message.AdaptHandler(message.PersistentConfig(handleAdvanceCeremonyState(l, ctx, marriageProcessor))),
	}
}

// handlePropose handles marriage proposal commands
func handlePropose(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.ProposeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.ProposeBody]) {
		l.WithFields(logrus.Fields{
			"type":             cmd.Type,
			"characterId":      cmd.CharacterId,
			"targetCharacterId": cmd.Body.TargetCharacterId,
		}).Debug("Processing marriage proposal command")
		
		if cmd.Type != marriage.CommandMarriagePropose {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the proposal
		proposal, err := processor.ProposeAndEmit(transactionId, cmd.CharacterId, cmd.Body.TargetCharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"proposerId": cmd.CharacterId,
				"targetId":   cmd.Body.TargetCharacterId,
			}).Error("Failed to process marriage proposal")
			
			// TODO: Emit error event
			return
		}
		
		l.WithFields(logrus.Fields{
			"proposalId": proposal.Id(),
			"proposerId": cmd.CharacterId,
			"targetId":   cmd.Body.TargetCharacterId,
		}).Info("Marriage proposal processed successfully")
	}
}

// handleAccept handles proposal acceptance commands
func handleAccept(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.AcceptBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.AcceptBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId": cmd.Body.ProposalId,
		}).Debug("Processing proposal acceptance command")
		
		if cmd.Type != marriage.CommandMarriageAccept {
			return
		}
		
		_ = uuid.New() // transactionId for future use
		
		// TODO: Implement AcceptProposalAndEmit method in processor
		// For now, just log the action
		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"characterId": cmd.CharacterId,
		}).Info("Proposal acceptance processed (implementation pending)")
	}
}

// handleDecline handles proposal decline commands
func handleDecline(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.DeclineBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.DeclineBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId": cmd.Body.ProposalId,
		}).Debug("Processing proposal decline command")
		
		if cmd.Type != marriage.CommandMarriageDecline {
			return
		}
		
		_ = uuid.New() // transactionId for future use
		
		// TODO: Implement DeclineProposalAndEmit method in processor
		// For now, just log the action
		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"characterId": cmd.CharacterId,
		}).Info("Proposal decline processed (implementation pending)")
	}
}

// handleCancel handles proposal cancellation commands
func handleCancel(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.CancelBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.CancelBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId": cmd.Body.ProposalId,
		}).Debug("Processing proposal cancellation command")
		
		if cmd.Type != marriage.CommandMarriageCancel {
			return
		}
		
		_ = uuid.New() // transactionId for future use
		
		// TODO: Implement CancelProposalAndEmit method in processor
		// For now, just log the action
		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"characterId": cmd.CharacterId,
		}).Info("Proposal cancellation processed (implementation pending)")
	}
}

// handleScheduleCeremony handles ceremony scheduling commands
func handleScheduleCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.ScheduleCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.ScheduleCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"marriageId":  cmd.Body.MarriageId,
			"scheduledAt": cmd.Body.ScheduledAt,
			"invitees":    len(cmd.Body.Invitees),
		}).Debug("Processing ceremony scheduling command")
		
		if cmd.Type != marriage.CommandCeremonySchedule {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony scheduling
		ceremony, err := processor.ScheduleCeremonyAndEmit(transactionId, cmd.Body.MarriageId, cmd.Body.ScheduledAt, cmd.Body.Invitees)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"marriageId":  cmd.Body.MarriageId,
				"scheduledAt": cmd.Body.ScheduledAt,
			}).Error("Failed to schedule ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"marriageId": cmd.Body.MarriageId,
		}).Info("Ceremony scheduled successfully")
	}
}

// handleStartCeremony handles ceremony start commands
func handleStartCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.StartCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.StartCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
		}).Debug("Processing ceremony start command")
		
		if cmd.Type != marriage.CommandCeremonyStart {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony start
		ceremony, err := processor.StartCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to start ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony started successfully")
	}
}

// handleCompleteCeremony handles ceremony completion commands
func handleCompleteCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.CompleteCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.CompleteCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
		}).Debug("Processing ceremony completion command")
		
		if cmd.Type != marriage.CommandCeremonyComplete {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony completion
		ceremony, err := processor.CompleteCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to complete ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony completed successfully")
	}
}

// handleCancelCeremony handles ceremony cancellation commands
func handleCancelCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.CancelCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.CancelCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
		}).Debug("Processing ceremony cancellation command")
		
		if cmd.Type != marriage.CommandCeremonyCancel {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony cancellation
		ceremony, err := processor.CancelCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to cancel ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony cancelled successfully")
	}
}

// handlePostponeCeremony handles ceremony postponement commands
func handlePostponeCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.PostponeCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.PostponeCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
		}).Debug("Processing ceremony postponement command")
		
		if cmd.Type != marriage.CommandCeremonyPostpone {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony postponement
		ceremony, err := processor.PostponeCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to postpone ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony postponed successfully")
	}
}

// handleRescheduleCeremony handles ceremony rescheduling commands
func handleRescheduleCeremony(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.RescheduleCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.RescheduleCeremonyBody]) {
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
			"scheduledAt": cmd.Body.ScheduledAt,
		}).Debug("Processing ceremony rescheduling command")
		
		if cmd.Type != marriage.CommandCeremonyReschedule {
			return
		}
		
		transactionId := uuid.New()
		
		// Process the ceremony rescheduling
		ceremony, err := processor.RescheduleCeremonyAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.ScheduledAt)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId":  cmd.Body.CeremonyId,
				"scheduledAt": cmd.Body.ScheduledAt,
			}).Error("Failed to reschedule ceremony")
			
			// TODO: Emit error event
			return
		}
		
		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony rescheduled successfully")
	}
}

// handleAddInvitee handles add invitee commands
func handleAddInvitee(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.AddInviteeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.AddInviteeBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
			"inviteeId":  cmd.Body.CharacterId,
		}).Debug("Processing add invitee command")
		
		if cmd.Type != marriage.CommandCeremonyAddInvitee {
			return
		}
		
		transactionId := uuid.New()
		
		// Process adding the invitee
		ceremony, err := processor.AddInviteeAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId": cmd.Body.CeremonyId,
				"inviteeId":  cmd.Body.CharacterId,
			}).Error("Failed to add invitee")
			
			// TODO: Emit error event
			return
		}
		
		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"inviteeId":  cmd.Body.CharacterId,
		}).Info("Invitee added successfully")
	}
}

// handleRemoveInvitee handles remove invitee commands
func handleRemoveInvitee(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.RemoveInviteeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.RemoveInviteeBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
			"inviteeId":  cmd.Body.CharacterId,
		}).Debug("Processing remove invitee command")
		
		if cmd.Type != marriage.CommandCeremonyRemoveInvitee {
			return
		}
		
		transactionId := uuid.New()
		
		// Process removing the invitee
		ceremony, err := processor.RemoveInviteeAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId": cmd.Body.CeremonyId,
				"inviteeId":  cmd.Body.CharacterId,
			}).Error("Failed to remove invitee")
			
			// TODO: Emit error event
			return
		}
		
		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"inviteeId":  cmd.Body.CharacterId,
		}).Info("Invitee removed successfully")
	}
}

// handleDivorce handles divorce commands
func handleDivorce(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.DivorceBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.DivorceBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"marriageId": cmd.Body.MarriageId,
		}).Debug("Processing divorce command")
		
		if cmd.Type != marriage.CommandMarriageDivorce {
			return
		}
		
		_ = uuid.New() // transactionId for future use
		
		// TODO: Implement DivorceAndEmit method in processor
		// For now, just log the action
		l.WithFields(logrus.Fields{
			"marriageId":  cmd.Body.MarriageId,
			"characterId": cmd.CharacterId,
		}).Info("Divorce processed (implementation pending)")
	}
}

// handleAdvanceCeremonyState handles ceremony state advancement commands
func handleAdvanceCeremonyState(l logrus.FieldLogger, ctx context.Context, processor marriageService.Processor) message.Handler[marriage.Command[marriage.AdvanceCeremonyStateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriage.Command[marriage.AdvanceCeremonyStateBody]) {
		l.WithFields(logrus.Fields{
			"type":       cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId": cmd.Body.CeremonyId,
			"nextState":  cmd.Body.NextState,
		}).Debug("Processing ceremony state advancement command")
		
		if cmd.Type != marriage.CommandCeremonyAdvanceState {
			return
		}
		
		_ = uuid.New() // transactionId for future use
		
		// TODO: Implement AdvanceCeremonyStateAndEmit method in processor
		// For now, just log the action
		l.WithFields(logrus.Fields{
			"ceremonyId": cmd.Body.CeremonyId,
			"nextState":  cmd.Body.NextState,
		}).Info("Ceremony state advancement processed (implementation pending)")
	}
}

// InitConsumers initializes the marriage command consumers
func InitConsumers(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			// Initialize consumer for marriage commands
			config := NewConfig(l)("marriage_commands")(marriage.EnvCommandTopic)(consumerGroupId)
			
			// Set up header parsers for tenant and span context
			rf(config,
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}