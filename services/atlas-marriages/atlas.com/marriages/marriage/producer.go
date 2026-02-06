package marriage

import (
	"time"

	"atlas-marriages/kafka/message/marriage"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// Proposal Event Producers

// ProposalCreatedEventProvider creates a provider for proposal created events
func ProposalCreatedEventProvider(proposalId uint32, proposerId uint32, targetCharacterId uint32, proposedAt time.Time, expiresAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(proposerId))
	value := &marriage.Event[marriage.ProposalCreatedBody]{
		CharacterId: proposerId,
		Type:        marriage.EventProposalCreated,
		Body: marriage.ProposalCreatedBody{
			ProposalId:        proposalId,
			ProposerId:        proposerId,
			TargetCharacterId: targetCharacterId,
			ProposedAt:        proposedAt,
			ExpiresAt:         expiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ProposalAcceptedEventProvider creates a provider for proposal accepted events
func ProposalAcceptedEventProvider(proposalId uint32, proposerId uint32, targetCharacterId uint32, acceptedAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(proposerId))
	value := &marriage.Event[marriage.ProposalAcceptedBody]{
		CharacterId: proposerId,
		Type:        marriage.EventProposalAccepted,
		Body: marriage.ProposalAcceptedBody{
			ProposalId:        proposalId,
			ProposerId:        proposerId,
			TargetCharacterId: targetCharacterId,
			AcceptedAt:        acceptedAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ProposalDeclinedEventProvider creates a provider for proposal declined events
func ProposalDeclinedEventProvider(proposalId uint32, proposerId uint32, targetCharacterId uint32, declinedAt time.Time, rejectionCount uint32, cooldownUntil time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(proposerId))
	value := &marriage.Event[marriage.ProposalDeclinedBody]{
		CharacterId: proposerId,
		Type:        marriage.EventProposalDeclined,
		Body: marriage.ProposalDeclinedBody{
			ProposalId:        proposalId,
			ProposerId:        proposerId,
			TargetCharacterId: targetCharacterId,
			DeclinedAt:        declinedAt,
			RejectionCount:    rejectionCount,
			CooldownUntil:     cooldownUntil,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ProposalExpiredEventProvider creates a provider for proposal expired events
func ProposalExpiredEventProvider(proposalId uint32, proposerId uint32, targetCharacterId uint32, expiredAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(proposerId))
	value := &marriage.Event[marriage.ProposalExpiredBody]{
		CharacterId: proposerId,
		Type:        marriage.EventProposalExpired,
		Body: marriage.ProposalExpiredBody{
			ProposalId:        proposalId,
			ProposerId:        proposerId,
			TargetCharacterId: targetCharacterId,
			ExpiredAt:         expiredAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ProposalCancelledEventProvider creates a provider for proposal cancelled events
func ProposalCancelledEventProvider(proposalId uint32, proposerId uint32, targetCharacterId uint32, cancelledAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(proposerId))
	value := &marriage.Event[marriage.ProposalCancelledBody]{
		CharacterId: proposerId,
		Type:        marriage.EventProposalCancelled,
		Body: marriage.ProposalCancelledBody{
			ProposalId:        proposalId,
			ProposerId:        proposerId,
			TargetCharacterId: targetCharacterId,
			CancelledAt:       cancelledAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// Marriage Event Producers

// MarriageCreatedEventProvider creates a provider for marriage created events
func MarriageCreatedEventProvider(marriageId uint32, characterId1 uint32, characterId2 uint32, marriedAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.MarriageCreatedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventMarriageCreated,
		Body: marriage.MarriageCreatedBody{
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			MarriedAt:    marriedAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MarriageDivorcedEventProvider creates a provider for marriage divorced events
func MarriageDivorcedEventProvider(marriageId uint32, characterId1 uint32, characterId2 uint32, divorcedAt time.Time, initiatedBy uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.MarriageDivorcedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventMarriageDivorced,
		Body: marriage.MarriageDivorcedBody{
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			DivorcedAt:   divorcedAt,
			InitiatedBy:  initiatedBy,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MarriageDeletedEventProvider creates a provider for marriage deleted events
func MarriageDeletedEventProvider(marriageId uint32, characterId1 uint32, characterId2 uint32, deletedAt time.Time, deletedBy uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.MarriageDeletedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventMarriageDeleted,
		Body: marriage.MarriageDeletedBody{
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			DeletedAt:    deletedAt,
			DeletedBy:    deletedBy,
			Reason:       reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// Ceremony Event Producers

// CeremonyScheduledEventProvider creates a provider for ceremony scheduled events
func CeremonyScheduledEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, scheduledAt time.Time, invitees []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyScheduledBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyScheduled,
		Body: marriage.CeremonyScheduledBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			ScheduledAt:  scheduledAt,
			Invitees:     invitees,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CeremonyStartedEventProvider creates a provider for ceremony started events
func CeremonyStartedEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, startedAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyStartedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyStarted,
		Body: marriage.CeremonyStartedBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			StartedAt:    startedAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CeremonyCompletedEventProvider creates a provider for ceremony completed events
func CeremonyCompletedEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, completedAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyCompletedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyCompleted,
		Body: marriage.CeremonyCompletedBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			CompletedAt:  completedAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CeremonyPostponedEventProvider creates a provider for ceremony postponed events
func CeremonyPostponedEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, postponedAt time.Time, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyPostponedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyPostponed,
		Body: marriage.CeremonyPostponedBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			PostponedAt:  postponedAt,
			Reason:       reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CeremonyCancelledEventProvider creates a provider for ceremony cancelled events
func CeremonyCancelledEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, cancelledAt time.Time, cancelledBy uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyCancelledBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyCancelled,
		Body: marriage.CeremonyCancelledBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			CancelledAt:  cancelledAt,
			CancelledBy:  cancelledBy,
			Reason:       reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CeremonyRescheduledEventProvider creates a provider for ceremony rescheduled events
func CeremonyRescheduledEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, rescheduledAt time.Time, newScheduledAt time.Time, rescheduledBy uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.CeremonyRescheduledBody]{
		CharacterId: characterId1,
		Type:        marriage.EventCeremonyRescheduled,
		Body: marriage.CeremonyRescheduledBody{
			CeremonyId:     ceremonyId,
			MarriageId:     marriageId,
			CharacterId1:   characterId1,
			CharacterId2:   characterId2,
			RescheduledAt:  rescheduledAt,
			NewScheduledAt: newScheduledAt,
			RescheduledBy:  rescheduledBy,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// Error Event Producers

// InviteeAddedEventProvider creates a provider for invitee added events
func InviteeAddedEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, inviteeId uint32, addedAt time.Time, addedBy uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.InviteeAddedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventInviteeAdded,
		Body: marriage.InviteeAddedBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			InviteeId:    inviteeId,
			AddedAt:      addedAt,
			AddedBy:      addedBy,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// InviteeRemovedEventProvider creates a provider for invitee removed events
func InviteeRemovedEventProvider(ceremonyId uint32, marriageId uint32, characterId1 uint32, characterId2 uint32, inviteeId uint32, removedAt time.Time, removedBy uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId1))
	value := &marriage.Event[marriage.InviteeRemovedBody]{
		CharacterId: characterId1,
		Type:        marriage.EventInviteeRemoved,
		Body: marriage.InviteeRemovedBody{
			CeremonyId:   ceremonyId,
			MarriageId:   marriageId,
			CharacterId1: characterId1,
			CharacterId2: characterId2,
			InviteeId:    inviteeId,
			RemovedAt:    removedAt,
			RemovedBy:    removedBy,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MarriageErrorEventProvider creates a provider for marriage error events
func MarriageErrorEventProvider(characterId uint32, errorType string, errorCode string, message string, context string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &marriage.Event[marriage.MarriageErrorBody]{
		CharacterId: characterId,
		Type:        marriage.EventMarriageError,
		Body: marriage.MarriageErrorBody{
			ErrorType:   errorType,
			ErrorCode:   errorCode,
			Message:     message,
			CharacterId: characterId,
			Context:     context,
			Timestamp:   time.Now(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
