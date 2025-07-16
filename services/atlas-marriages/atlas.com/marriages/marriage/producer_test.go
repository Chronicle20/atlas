package marriage

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-kafka/producer"
)

func TestProposalCreatedEventProvider(t *testing.T) {
	proposalId := uint32(1)
	proposerId := uint32(100)
	targetCharacterId := uint32(200)
	proposedAt := time.Now()
	expiresAt := proposedAt.Add(24 * time.Hour)

	provider := ProposalCreatedEventProvider(proposalId, proposerId, targetCharacterId, proposedAt, expiresAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(proposerId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
	
	// Verify the message is properly structured
	if msg.Value == nil {
		t.Error("Expected message value to be set")
	}
}

func TestProposalAcceptedEventProvider(t *testing.T) {
	proposalId := uint32(1)
	proposerId := uint32(100)
	targetCharacterId := uint32(200)
	acceptedAt := time.Now()

	provider := ProposalAcceptedEventProvider(proposalId, proposerId, targetCharacterId, acceptedAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(proposerId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestProposalDeclinedEventProvider(t *testing.T) {
	proposalId := uint32(1)
	proposerId := uint32(100)
	targetCharacterId := uint32(200)
	declinedAt := time.Now()
	rejectionCount := uint32(2)
	cooldownUntil := declinedAt.Add(48 * time.Hour)

	provider := ProposalDeclinedEventProvider(proposalId, proposerId, targetCharacterId, declinedAt, rejectionCount, cooldownUntil)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(proposerId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestProposalExpiredEventProvider(t *testing.T) {
	proposalId := uint32(1)
	proposerId := uint32(100)
	targetCharacterId := uint32(200)
	expiredAt := time.Now()

	provider := ProposalExpiredEventProvider(proposalId, proposerId, targetCharacterId, expiredAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(proposerId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestProposalCancelledEventProvider(t *testing.T) {
	proposalId := uint32(1)
	proposerId := uint32(100)
	targetCharacterId := uint32(200)
	cancelledAt := time.Now()

	provider := ProposalCancelledEventProvider(proposalId, proposerId, targetCharacterId, cancelledAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(proposerId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestMarriageCreatedEventProvider(t *testing.T) {
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	marriedAt := time.Now()

	provider := MarriageCreatedEventProvider(marriageId, characterId1, characterId2, marriedAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestMarriageDivorcedEventProvider(t *testing.T) {
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	divorcedAt := time.Now()
	initiator := uint32(100)

	provider := MarriageDivorcedEventProvider(marriageId, characterId1, characterId2, divorcedAt, initiator)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestMarriageDeletedEventProvider(t *testing.T) {
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	deletedAt := time.Now()
	deletedBy := uint32(100)
	reason := "character_deleted"

	provider := MarriageDeletedEventProvider(marriageId, characterId1, characterId2, deletedAt, deletedBy, reason)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyScheduledEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	scheduledAt := time.Now()
	invitees := []uint32{300, 400}

	provider := CeremonyScheduledEventProvider(ceremonyId, marriageId, characterId1, characterId2, scheduledAt, invitees)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyStartedEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	startedAt := time.Now()

	provider := CeremonyStartedEventProvider(ceremonyId, marriageId, characterId1, characterId2, startedAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyCompletedEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	completedAt := time.Now()

	provider := CeremonyCompletedEventProvider(ceremonyId, marriageId, characterId1, characterId2, completedAt)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyPostponedEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	postponedAt := time.Now()
	reason := "partner_disconnected"

	provider := CeremonyPostponedEventProvider(ceremonyId, marriageId, characterId1, characterId2, postponedAt, reason)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyCancelledEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	cancelledAt := time.Now()
	cancelledBy := uint32(100)
	reason := "manual_cancellation"

	provider := CeremonyCancelledEventProvider(ceremonyId, marriageId, characterId1, characterId2, cancelledAt, cancelledBy, reason)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestCeremonyRescheduledEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	rescheduledAt := time.Now()
	newScheduledAt := time.Now().Add(24 * time.Hour)
	rescheduledBy := uint32(100)

	provider := CeremonyRescheduledEventProvider(ceremonyId, marriageId, characterId1, characterId2, rescheduledAt, newScheduledAt, rescheduledBy)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestInviteeAddedEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	inviteeCharacterId := uint32(300)
	addedAt := time.Now()
	addedBy := uint32(100)

	provider := InviteeAddedEventProvider(ceremonyId, marriageId, characterId1, characterId2, inviteeCharacterId, addedAt, addedBy)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestInviteeRemovedEventProvider(t *testing.T) {
	ceremonyId := uint32(1)
	marriageId := uint32(1)
	characterId1 := uint32(100)
	characterId2 := uint32(200)
	inviteeCharacterId := uint32(300)
	removedAt := time.Now()
	removedBy := uint32(100)

	provider := InviteeRemovedEventProvider(ceremonyId, marriageId, characterId1, characterId2, inviteeCharacterId, removedAt, removedBy)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId1))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}

func TestMarriageErrorEventProvider(t *testing.T) {
	characterId := uint32(100)
	errorType := "invalid_proposal"
	errorCode := "SELF_PROPOSAL"
	errorMessage := "Cannot propose to yourself"
	context := "proposal_validation"

	provider := MarriageErrorEventProvider(characterId, errorType, errorCode, errorMessage, context)
	
	messages, err := provider()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	
	msg := messages[0]
	expectedKey := producer.CreateKey(int(characterId))
	if string(msg.Key) != string(expectedKey) {
		t.Errorf("Expected key %s, got %s", expectedKey, msg.Key)
	}
}