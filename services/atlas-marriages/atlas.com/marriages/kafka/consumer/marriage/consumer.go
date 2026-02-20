package marriage

import (
	"context"

	localConsumer "atlas-marriages/kafka/consumer"
	"atlas-marriages/kafka/message"
	marriageMsg "atlas-marriages/kafka/message/marriage"
	"atlas-marriages/kafka/producer"
	marriageService "atlas-marriages/marriage"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafka "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
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
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(marriageMsg.EnvCommandTopic)()
			// Proposal command handlers
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handlePropose(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleAccept(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleDecline(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleCancel(marriageService.NewProcessor, db)))); err != nil {
				return err
			}

			// Ceremony command handlers
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleScheduleCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleStartCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleCompleteCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleCancelCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handlePostponeCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleRescheduleCeremony(marriageService.NewProcessor, db)))); err != nil {
				return err
			}

			// Invitee command handlers
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleAddInvitee(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleRemoveInvitee(marriageService.NewProcessor, db)))); err != nil {
				return err
			}

			// Divorce command handler
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleDivorce(marriageService.NewProcessor, db)))); err != nil {
				return err
			}

			// Advance ceremony state handler
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleAdvanceCeremonyState(marriageService.NewProcessor, db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handlePropose handles marriage proposal commands
func handlePropose(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.ProposeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.ProposeBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":              cmd.Type,
			"characterId":       cmd.CharacterId,
			"targetCharacterId": cmd.Body.TargetCharacterId,
		}).Debug("Processing marriage proposal command")

		if cmd.Type != marriageMsg.CommandMarriagePropose {
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

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"PROPOSAL_FAILED",
				"MARRIAGE_PROPOSAL_ERROR",
				err.Error(),
				"marriage_proposal",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for proposal failure")
			}
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
func handleAccept(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.AcceptBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.AcceptBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId":  cmd.Body.ProposalId,
		}).Debug("Processing proposal acceptance command")

		if cmd.Type != marriageMsg.CommandMarriageAccept {
			return
		}

		transactionId := uuid.New()

		// Process the proposal acceptance
		marriage, err := processor.AcceptProposalAndEmit(transactionId, cmd.Body.ProposalId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"proposalId":  cmd.Body.ProposalId,
				"characterId": cmd.CharacterId,
			}).Error("Failed to process proposal acceptance")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"PROPOSAL_ACCEPT_FAILED",
				"MARRIAGE_PROPOSAL_ACCEPT_ERROR",
				err.Error(),
				"marriage_proposal_accept",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for proposal acceptance failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"marriageId":  marriage.Id(),
			"characterId": cmd.CharacterId,
		}).Info("Proposal acceptance processed successfully")
	}
}

// handleDecline handles proposal decline commands
func handleDecline(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.DeclineBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.DeclineBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId":  cmd.Body.ProposalId,
		}).Debug("Processing proposal decline command")

		if cmd.Type != marriageMsg.CommandMarriageDecline {
			return
		}

		transactionId := uuid.New()

		// Process the proposal decline
		_, err := processor.DeclineProposalAndEmit(transactionId, cmd.Body.ProposalId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"proposalId":  cmd.Body.ProposalId,
				"characterId": cmd.CharacterId,
			}).Error("Failed to process proposal decline")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"PROPOSAL_DECLINE_FAILED",
				"MARRIAGE_PROPOSAL_DECLINE_ERROR",
				err.Error(),
				"marriage_proposal_decline",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for proposal decline failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"characterId": cmd.CharacterId,
		}).Info("Proposal decline processed successfully")
	}
}

// handleCancel handles proposal cancellation commands
func handleCancel(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.CancelBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.CancelBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"proposalId":  cmd.Body.ProposalId,
		}).Debug("Processing proposal cancellation command")

		if cmd.Type != marriageMsg.CommandMarriageCancel {
			return
		}

		transactionId := uuid.New()

		// Process the proposal cancellation
		_, err := processor.CancelProposalAndEmit(transactionId, cmd.Body.ProposalId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"proposalId":  cmd.Body.ProposalId,
				"characterId": cmd.CharacterId,
			}).Error("Failed to process proposal cancellation")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"PROPOSAL_CANCEL_FAILED",
				"MARRIAGE_PROPOSAL_CANCEL_ERROR",
				err.Error(),
				"marriage_proposal_cancel",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for proposal cancellation failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"proposalId":  cmd.Body.ProposalId,
			"characterId": cmd.CharacterId,
		}).Info("Proposal cancellation processed successfully")
	}
}

// handleScheduleCeremony handles ceremony scheduling commands
func handleScheduleCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.ScheduleCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.ScheduleCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"marriageId":  cmd.Body.MarriageId,
			"scheduledAt": cmd.Body.ScheduledAt,
			"invitees":    len(cmd.Body.Invitees),
		}).Debug("Processing ceremony scheduling command")

		if cmd.Type != marriageMsg.CommandCeremonySchedule {
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

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_SCHEDULE_FAILED",
				"CEREMONY_SCHEDULE_ERROR",
				err.Error(),
				"ceremony_schedule",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony schedule failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"marriageId": cmd.Body.MarriageId,
		}).Info("Ceremony scheduled successfully")
	}
}

// handleStartCeremony handles ceremony start commands
func handleStartCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.StartCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.StartCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
		}).Debug("Processing ceremony start command")

		if cmd.Type != marriageMsg.CommandCeremonyStart {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony start
		ceremony, err := processor.StartCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to start ceremony")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_START_FAILED",
				"CEREMONY_START_ERROR",
				err.Error(),
				"ceremony_start",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony start failure")
			}
			return
		}

		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony started successfully")
	}
}

// handleCompleteCeremony handles ceremony completion commands
func handleCompleteCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.CompleteCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.CompleteCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
		}).Debug("Processing ceremony completion command")

		if cmd.Type != marriageMsg.CommandCeremonyComplete {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony completion
		ceremony, err := processor.CompleteCeremonyAndEmit(transactionId, cmd.Body.CeremonyId)
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to complete ceremony")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_COMPLETE_FAILED",
				"CEREMONY_COMPLETE_ERROR",
				err.Error(),
				"ceremony_complete",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony complete failure")
			}
			return
		}

		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony completed successfully")
	}
}

// handleCancelCeremony handles ceremony cancellation commands
func handleCancelCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.CancelCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.CancelCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
		}).Debug("Processing ceremony cancellation command")

		if cmd.Type != marriageMsg.CommandCeremonyCancel {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony cancellation
		ceremony, err := processor.CancelCeremonyAndEmit(transactionId, cmd.Body.CeremonyId, cmd.CharacterId, "ceremony_cancelled")
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to cancel ceremony")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_CANCEL_FAILED",
				"CEREMONY_CANCEL_ERROR",
				err.Error(),
				"ceremony_cancel",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony cancel failure")
			}
			return
		}

		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony cancelled successfully")
	}
}

// handlePostponeCeremony handles ceremony postponement commands
func handlePostponeCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.PostponeCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.PostponeCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
		}).Debug("Processing ceremony postponement command")

		if cmd.Type != marriageMsg.CommandCeremonyPostpone {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony postponement
		ceremony, err := processor.PostponeCeremonyAndEmit(transactionId, cmd.Body.CeremonyId, "ceremony_postponed")
		if err != nil {
			l.WithError(err).WithField("ceremonyId", cmd.Body.CeremonyId).Error("Failed to postpone ceremony")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_POSTPONE_FAILED",
				"CEREMONY_POSTPONE_ERROR",
				err.Error(),
				"ceremony_postpone",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony postpone failure")
			}
			return
		}

		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony postponed successfully")
	}
}

// handleRescheduleCeremony handles ceremony rescheduling commands
func handleRescheduleCeremony(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.RescheduleCeremonyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.RescheduleCeremonyBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
			"scheduledAt": cmd.Body.ScheduledAt,
		}).Debug("Processing ceremony rescheduling command")

		if cmd.Type != marriageMsg.CommandCeremonyReschedule {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony rescheduling
		ceremony, err := processor.RescheduleCeremonyAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.ScheduledAt, cmd.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId":  cmd.Body.CeremonyId,
				"scheduledAt": cmd.Body.ScheduledAt,
			}).Error("Failed to reschedule ceremony")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_RESCHEDULE_FAILED",
				"CEREMONY_RESCHEDULE_ERROR",
				err.Error(),
				"ceremony_reschedule",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony reschedule failure")
			}
			return
		}

		l.WithField("ceremonyId", ceremony.Id()).Info("Ceremony rescheduled successfully")
	}
}

// handleAddInvitee handles add invitee commands
func handleAddInvitee(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.AddInviteeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.AddInviteeBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
			"inviteeId":   cmd.Body.CharacterId,
		}).Debug("Processing add invitee command")

		if cmd.Type != marriageMsg.CommandCeremonyAddInvitee {
			return
		}

		transactionId := uuid.New()

		// Process adding the invitee
		ceremony, err := processor.AddInviteeAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.CharacterId, cmd.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId": cmd.Body.CeremonyId,
				"inviteeId":  cmd.Body.CharacterId,
			}).Error("Failed to add invitee")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"INVITEE_ADD_FAILED",
				"INVITEE_ADD_ERROR",
				err.Error(),
				"invitee_add",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for invitee add failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"inviteeId":  cmd.Body.CharacterId,
		}).Info("Invitee added successfully")
	}
}

// handleRemoveInvitee handles remove invitee commands
func handleRemoveInvitee(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.RemoveInviteeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.RemoveInviteeBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
			"inviteeId":   cmd.Body.CharacterId,
		}).Debug("Processing remove invitee command")

		if cmd.Type != marriageMsg.CommandCeremonyRemoveInvitee {
			return
		}

		transactionId := uuid.New()

		// Process removing the invitee
		ceremony, err := processor.RemoveInviteeAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.CharacterId, cmd.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId": cmd.Body.CeremonyId,
				"inviteeId":  cmd.Body.CharacterId,
			}).Error("Failed to remove invitee")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"INVITEE_REMOVE_FAILED",
				"INVITEE_REMOVE_ERROR",
				err.Error(),
				"invitee_remove",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for invitee remove failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"inviteeId":  cmd.Body.CharacterId,
		}).Info("Invitee removed successfully")
	}
}

// handleDivorce handles divorce commands
func handleDivorce(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.DivorceBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.DivorceBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"marriageId":  cmd.Body.MarriageId,
		}).Debug("Processing divorce command")

		if cmd.Type != marriageMsg.CommandMarriageDivorce {
			return
		}

		transactionId := uuid.New()

		// Process the divorce
		marriage, err := processor.DivorceAndEmit(transactionId, cmd.Body.MarriageId, cmd.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"marriageId":  cmd.Body.MarriageId,
				"characterId": cmd.CharacterId,
			}).Error("Failed to process divorce")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"DIVORCE_FAILED",
				"MARRIAGE_DIVORCE_ERROR",
				err.Error(),
				"marriage_divorce",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for divorce failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"marriageId":  marriage.Id(),
			"characterId": cmd.CharacterId,
		}).Info("Divorce processed successfully")
	}
}

// handleAdvanceCeremonyState handles ceremony state advancement commands
func handleAdvanceCeremonyState(pp marriageService.ProcessorProducer, db *gorm.DB) kafka.Handler[marriageMsg.Command[marriageMsg.AdvanceCeremonyStateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd marriageMsg.Command[marriageMsg.AdvanceCeremonyStateBody]) {
		processor := pp(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        cmd.Type,
			"characterId": cmd.CharacterId,
			"ceremonyId":  cmd.Body.CeremonyId,
			"nextState":   cmd.Body.NextState,
		}).Debug("Processing ceremony state advancement command")

		if cmd.Type != marriageMsg.CommandCeremonyAdvanceState {
			return
		}

		transactionId := uuid.New()

		// Process the ceremony state advancement
		ceremony, err := processor.AdvanceCeremonyStateAndEmit(transactionId, cmd.Body.CeremonyId, cmd.Body.NextState)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"ceremonyId": cmd.Body.CeremonyId,
				"nextState":  cmd.Body.NextState,
			}).Error("Failed to advance ceremony state")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				cmd.CharacterId,
				"CEREMONY_STATE_ADVANCE_FAILED",
				"CEREMONY_STATE_ADVANCE_ERROR",
				err.Error(),
				"ceremony_state_advance",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put(marriageMsg.EnvEventTopicStatus, errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for ceremony state advance failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"nextState":  cmd.Body.NextState,
		}).Info("Ceremony state advancement processed successfully")
	}
}

// InitConsumers initializes the marriage command consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			// Initialize consumer for marriage commands
			config := NewConfig(l)("marriage_commands")(marriageMsg.EnvCommandTopic)(consumerGroupId)

			// Set up header parsers for tenant and span context
			rf(config,
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}
