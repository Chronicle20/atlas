package marriage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProposal_Creation(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	proposal, err := NewProposalBuilder(proposerId, targetId, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	if proposal.ProposerId() != proposerId {
		t.Errorf("Expected proposer ID %d, got %d", proposerId, proposal.ProposerId())
	}

	if proposal.TargetId() != targetId {
		t.Errorf("Expected target ID %d, got %d", targetId, proposal.TargetId())
	}

	if proposal.Status() != ProposalStatusPending {
		t.Errorf("Expected status %s, got %s", ProposalStatusPending, proposal.Status())
	}

	if !proposal.IsPending() {
		t.Error("Expected proposal to be pending")
	}

	if proposal.IsExpired() {
		t.Error("Expected proposal to not be expired")
	}

	if proposal.RejectionCount() != 0 {
		t.Errorf("Expected rejection count 0, got %d", proposal.RejectionCount())
	}
}

func TestProposal_Accept(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	proposal, err := NewProposalBuilder(proposerId, targetId, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	acceptedProposal, err := proposal.Accept()
	if err != nil {
		t.Fatalf("Failed to accept proposal: %v", err)
	}

	if acceptedProposal.Status() != ProposalStatusAccepted {
		t.Errorf("Expected status %s, got %s", ProposalStatusAccepted, acceptedProposal.Status())
	}

	if !acceptedProposal.IsAccepted() {
		t.Error("Expected proposal to be accepted")
	}

	if acceptedProposal.RespondedAt() == nil {
		t.Error("Expected responded at timestamp to be set")
	}
}

func TestProposal_Reject(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	proposal, err := NewProposalBuilder(proposerId, targetId, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	rejectedProposal, err := proposal.Reject()
	if err != nil {
		t.Fatalf("Failed to reject proposal: %v", err)
	}

	if rejectedProposal.Status() != ProposalStatusRejected {
		t.Errorf("Expected status %s, got %s", ProposalStatusRejected, rejectedProposal.Status())
	}

	if !rejectedProposal.IsRejected() {
		t.Error("Expected proposal to be rejected")
	}

	if rejectedProposal.RespondedAt() == nil {
		t.Error("Expected responded at timestamp to be set")
	}

	if rejectedProposal.RejectionCount() != 1 {
		t.Errorf("Expected rejection count 1, got %d", rejectedProposal.RejectionCount())
	}

	if rejectedProposal.CooldownUntil() == nil {
		t.Error("Expected cooldown until timestamp to be set")
	}
}

func TestProposal_CooldownCalculation(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	// Test initial cooldown
	proposal, err := NewProposalBuilder(proposerId, targetId, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	initialCooldown := proposal.CalculateNextCooldown()
	if initialCooldown != InitialPerTargetCooldown {
		t.Errorf("Expected initial cooldown %v, got %v", InitialPerTargetCooldown, initialCooldown)
	}

	// Test exponential backoff
	proposal, err = proposal.Builder().SetRejectionCount(1).Build()
	if err != nil {
		t.Fatalf("Failed to build proposal with rejection count: %v", err)
	}
	secondCooldown := proposal.CalculateNextCooldown()
	expectedSecondCooldown := 2 * InitialPerTargetCooldown
	if secondCooldown != expectedSecondCooldown {
		t.Errorf("Expected second cooldown %v, got %v", expectedSecondCooldown, secondCooldown)
	}

	// Test further exponential backoff
	proposal, err = proposal.Builder().SetRejectionCount(2).Build()
	if err != nil {
		t.Fatalf("Failed to build proposal with rejection count: %v", err)
	}
	thirdCooldown := proposal.CalculateNextCooldown()
	expectedThirdCooldown := 4 * InitialPerTargetCooldown
	if thirdCooldown != expectedThirdCooldown {
		t.Errorf("Expected third cooldown %v, got %v", expectedThirdCooldown, thirdCooldown)
	}
}

func TestProposal_Expiry(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	// Create a proposal that expires in the past
	pastTime := time.Now().Add(-25 * time.Hour) // 25 hours ago, older than ProposalExpiryDuration (24h)
	proposal, err := NewProposalBuilder(proposerId, targetId, tenantId).
		SetProposedAt(pastTime).
		SetExpiresAt(pastTime.Add(ProposalExpiryDuration)). // This will still be in the past
		Build()
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	if !proposal.IsExpired() {
		t.Error("Expected proposal to be expired")
	}

	if proposal.CanRespond() {
		t.Error("Expected expired proposal to not be respondable")
	}
}

func TestProposal_ValidationRules(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)

	// Test self-proposal validation
	_, err := NewProposalBuilder(proposerId, proposerId, tenantId).Build()
	if err == nil {
		t.Error("Expected error for self-proposal")
	}

	// Test zero proposer ID
	_, err = NewProposalBuilder(0, targetId, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero proposer ID")
	}

	// Test zero target ID
	_, err = NewProposalBuilder(proposerId, 0, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero target ID")
	}

	// Test nil tenant ID
	_, err = NewProposalBuilder(proposerId, targetId, uuid.Nil).Build()
	if err == nil {
		t.Error("Expected error for nil tenant ID")
	}
}