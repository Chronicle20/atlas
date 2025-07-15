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

func TestProposal_StateTransitionValidation(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)
	now := time.Now()

	tests := []struct {
		name        string
		status      ProposalStatus
		respondedAt *time.Time
		cooldownUntil *time.Time
		expectError bool
		errorContains string
	}{
		{
			name:        "pending proposal with response timestamp",
			status:      ProposalStatusPending,
			respondedAt: &now,
			expectError: true,
			errorContains: "pending proposal cannot have response timestamp",
		},
		{
			name:        "accepted proposal without response timestamp",
			status:      ProposalStatusAccepted,
			respondedAt: nil,
			expectError: true,
			errorContains: "accepted proposal must have response timestamp",
		},
		{
			name:        "rejected proposal without response timestamp",
			status:      ProposalStatusRejected,
			respondedAt: nil,
			expectError: true,
			errorContains: "rejected proposal must have response timestamp",
		},
		{
			name:        "rejected proposal without cooldown timestamp",
			status:      ProposalStatusRejected,
			respondedAt: &now,
			cooldownUntil: nil,
			expectError: true,
			errorContains: "rejected proposal must have cooldown timestamp",
		},
		{
			name:        "expired proposal with response timestamp",
			status:      ProposalStatusExpired,
			respondedAt: &now,
			expectError: true,
			errorContains: "expired proposal cannot have response timestamp",
		},
		{
			name:        "cancelled proposal with response timestamp",
			status:      ProposalStatusCancelled,
			respondedAt: &now,
			expectError: true,
			errorContains: "cancelled proposal cannot have response timestamp",
		},
		{
			name:        "valid pending proposal",
			status:      ProposalStatusPending,
			respondedAt: nil,
			expectError: false,
		},
		{
			name:        "valid accepted proposal",
			status:      ProposalStatusAccepted,
			respondedAt: &now,
			expectError: false,
		},
		{
			name:        "valid rejected proposal",
			status:      ProposalStatusRejected,
			respondedAt: &now,
			cooldownUntil: &now,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewProposalBuilder(proposerId, targetId, tenantId).
				SetStatus(tt.status).
				SetRespondedAt(tt.respondedAt).
				SetCooldownUntil(tt.cooldownUntil)

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s", tt.name)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestProposal_ExpiryTimeValidation(t *testing.T) {
	tenantId := uuid.New()
	proposerId := uint32(1)
	targetId := uint32(2)
	now := time.Now()

	// Test expiry time before proposal time
	_, err := NewProposalBuilder(proposerId, targetId, tenantId).
		SetProposedAt(now).
		SetExpiresAt(now.Add(-1 * time.Hour)).
		Build()
	if err == nil {
		t.Error("Expected error for expiry time before proposal time")
	}
	if !contains(err.Error(), "expiry time cannot be before proposal time") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

// Marriage model tests
func TestMarriage_Creation(t *testing.T) {
	tenantId := uuid.New()
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	marriage, err := NewBuilder(characterId1, characterId2, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create marriage: %v", err)
	}

	if marriage.CharacterId1() != characterId1 {
		t.Errorf("Expected character ID 1 %d, got %d", characterId1, marriage.CharacterId1())
	}

	if marriage.CharacterId2() != characterId2 {
		t.Errorf("Expected character ID 2 %d, got %d", characterId2, marriage.CharacterId2())
	}

	if marriage.Status() != StatusProposed {
		t.Errorf("Expected status %s, got %s", StatusProposed, marriage.Status())
	}

	if marriage.IsActive() {
		t.Error("Expected marriage to not be active")
	}

	if marriage.IsExpired() {
		t.Error("Expected marriage to not be expired")
	}

	if marriage.IsDivorced() {
		t.Error("Expected marriage to not be divorced")
	}

	if !marriage.CanAccept() {
		t.Error("Expected marriage to be acceptable")
	}
}

func TestMarriage_ValidationRules(t *testing.T) {
	tenantId := uuid.New()
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	// Test self-marriage validation
	_, err := NewBuilder(characterId1, characterId1, tenantId).Build()
	if err == nil {
		t.Error("Expected error for self-marriage")
	}
	if !contains(err.Error(), "character cannot marry themselves") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test zero character ID 1
	_, err = NewBuilder(0, characterId2, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero character ID 1")
	}
	if !contains(err.Error(), "character ID 1 is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test zero character ID 2
	_, err = NewBuilder(characterId1, 0, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero character ID 2")
	}
	if !contains(err.Error(), "character ID 2 is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test nil tenant ID
	_, err = NewBuilder(characterId1, characterId2, uuid.Nil).Build()
	if err == nil {
		t.Error("Expected error for nil tenant ID")
	}
	if !contains(err.Error(), "tenant ID is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestMarriage_StateTransitionValidation(t *testing.T) {
	tenantId := uuid.New()
	characterId1 := uint32(1)
	characterId2 := uint32(2)
	now := time.Now()

	tests := []struct {
		name        string
		status      MarriageStatus
		engagedAt   *time.Time
		marriedAt   *time.Time
		divorcedAt  *time.Time
		expectError bool
		errorContains string
	}{
		{
			name:        "proposed marriage with engagement timestamp",
			status:      StatusProposed,
			engagedAt:   &now,
			expectError: true,
			errorContains: "proposed marriage cannot have engagement timestamp",
		},
		{
			name:        "proposed marriage with marriage timestamp",
			status:      StatusProposed,
			marriedAt:   &now,
			expectError: true,
			errorContains: "proposed marriage cannot have marriage timestamp",
		},
		{
			name:        "proposed marriage with divorce timestamp",
			status:      StatusProposed,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "proposed marriage cannot have divorce timestamp",
		},
		{
			name:        "engaged marriage without engagement timestamp",
			status:      StatusEngaged,
			engagedAt:   nil,
			expectError: true,
			errorContains: "engaged marriage must have engagement timestamp",
		},
		{
			name:        "engaged marriage with marriage timestamp",
			status:      StatusEngaged,
			engagedAt:   &now,
			marriedAt:   &now,
			expectError: true,
			errorContains: "engaged marriage cannot have marriage timestamp",
		},
		{
			name:        "engaged marriage with divorce timestamp",
			status:      StatusEngaged,
			engagedAt:   &now,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "engaged marriage cannot have divorce timestamp",
		},
		{
			name:        "married marriage without engagement timestamp",
			status:      StatusMarried,
			engagedAt:   nil,
			marriedAt:   &now,
			expectError: true,
			errorContains: "married marriage must have engagement timestamp",
		},
		{
			name:        "married marriage without marriage timestamp",
			status:      StatusMarried,
			engagedAt:   &now,
			marriedAt:   nil,
			expectError: true,
			errorContains: "married marriage must have marriage timestamp",
		},
		{
			name:        "married marriage with divorce timestamp",
			status:      StatusMarried,
			engagedAt:   &now,
			marriedAt:   &now,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "married marriage cannot have divorce timestamp",
		},
		{
			name:        "divorced marriage without engagement timestamp",
			status:      StatusDivorced,
			engagedAt:   nil,
			marriedAt:   &now,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "divorced marriage must have engagement timestamp",
		},
		{
			name:        "divorced marriage without marriage timestamp",
			status:      StatusDivorced,
			engagedAt:   &now,
			marriedAt:   nil,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "divorced marriage must have marriage timestamp",
		},
		{
			name:        "divorced marriage without divorce timestamp",
			status:      StatusDivorced,
			engagedAt:   &now,
			marriedAt:   &now,
			divorcedAt:  nil,
			expectError: true,
			errorContains: "divorced marriage must have divorce timestamp",
		},
		{
			name:        "expired marriage with engagement timestamp",
			status:      StatusExpired,
			engagedAt:   &now,
			expectError: true,
			errorContains: "expired marriage cannot have engagement timestamp",
		},
		{
			name:        "expired marriage with marriage timestamp",
			status:      StatusExpired,
			marriedAt:   &now,
			expectError: true,
			errorContains: "expired marriage cannot have marriage timestamp",
		},
		{
			name:        "expired marriage with divorce timestamp",
			status:      StatusExpired,
			divorcedAt:  &now,
			expectError: true,
			errorContains: "expired marriage cannot have divorce timestamp",
		},
		{
			name:        "valid proposed marriage",
			status:      StatusProposed,
			expectError: false,
		},
		{
			name:        "valid engaged marriage",
			status:      StatusEngaged,
			engagedAt:   &now,
			expectError: false,
		},
		{
			name:        "valid married marriage",
			status:      StatusMarried,
			engagedAt:   &now,
			marriedAt:   &now,
			expectError: false,
		},
		{
			name:        "valid divorced marriage",
			status:      StatusDivorced,
			engagedAt:   &now,
			marriedAt:   &now,
			divorcedAt:  &now,
			expectError: false,
		},
		{
			name:        "valid expired marriage",
			status:      StatusExpired,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(characterId1, characterId2, tenantId).
				SetStatus(tt.status).
				SetEngagedAt(tt.engagedAt).
				SetMarriedAt(tt.marriedAt).
				SetDivorcedAt(tt.divorcedAt)

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s", tt.name)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestMarriage_BusinessLogic(t *testing.T) {
	tenantId := uuid.New()
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	marriage, err := NewBuilder(characterId1, characterId2, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create marriage: %v", err)
	}

	// Test partner identification
	if !marriage.IsPartner(characterId1) {
		t.Error("Expected character 1 to be a partner")
	}
	if !marriage.IsPartner(characterId2) {
		t.Error("Expected character 2 to be a partner")
	}
	if marriage.IsPartner(uint32(3)) {
		t.Error("Expected character 3 to not be a partner")
	}

	// Test proposer identification
	if !marriage.IsProposer(characterId1) {
		t.Error("Expected character 1 to be the proposer")
	}
	if marriage.IsProposer(characterId2) {
		t.Error("Expected character 2 to not be the proposer")
	}

	// Test get partner
	partner, found := marriage.GetPartner(characterId1)
	if !found {
		t.Error("Expected to find partner for character 1")
	}
	if partner != characterId2 {
		t.Errorf("Expected partner %d, got %d", characterId2, partner)
	}

	partner, found = marriage.GetPartner(characterId2)
	if !found {
		t.Error("Expected to find partner for character 2")
	}
	if partner != characterId1 {
		t.Errorf("Expected partner %d, got %d", characterId1, partner)
	}

	_, found = marriage.GetPartner(uint32(3))
	if found {
		t.Error("Expected to not find partner for character 3")
	}

	// Test state transitions
	acceptedMarriage, err := marriage.Accept()
	if err != nil {
		t.Fatalf("Failed to accept marriage: %v", err)
	}
	if acceptedMarriage.Status() != StatusEngaged {
		t.Errorf("Expected status %s, got %s", StatusEngaged, acceptedMarriage.Status())
	}
	if !acceptedMarriage.CanMarry() {
		t.Error("Expected engaged marriage to be marriageable")
	}

	marriedMarriage, err := acceptedMarriage.Marry()
	if err != nil {
		t.Fatalf("Failed to marry: %v", err)
	}
	if marriedMarriage.Status() != StatusMarried {
		t.Errorf("Expected status %s, got %s", StatusMarried, marriedMarriage.Status())
	}
	if !marriedMarriage.IsActive() {
		t.Error("Expected married marriage to be active")
	}
	if !marriedMarriage.CanDivorce() {
		t.Error("Expected married marriage to be divorceable")
	}

	divorcedMarriage, err := marriedMarriage.Divorce()
	if err != nil {
		t.Fatalf("Failed to divorce: %v", err)
	}
	if divorcedMarriage.Status() != StatusDivorced {
		t.Errorf("Expected status %s, got %s", StatusDivorced, divorcedMarriage.Status())
	}
	if !divorcedMarriage.IsDivorced() {
		t.Error("Expected divorced marriage to be divorced")
	}

	// Test expiry
	expiredMarriage, err := marriage.Expire()
	if err != nil {
		t.Fatalf("Failed to expire marriage: %v", err)
	}
	if expiredMarriage.Status() != StatusExpired {
		t.Errorf("Expected status %s, got %s", StatusExpired, expiredMarriage.Status())
	}
	if !expiredMarriage.IsExpired() {
		t.Error("Expected expired marriage to be expired")
	}
}

// Ceremony model tests
func TestCeremony_Creation(t *testing.T) {
	tenantId := uuid.New()
	marriageId := uint32(1)
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	ceremony, err := NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create ceremony: %v", err)
	}

	if ceremony.MarriageId() != marriageId {
		t.Errorf("Expected marriage ID %d, got %d", marriageId, ceremony.MarriageId())
	}

	if ceremony.CharacterId1() != characterId1 {
		t.Errorf("Expected character ID 1 %d, got %d", characterId1, ceremony.CharacterId1())
	}

	if ceremony.CharacterId2() != characterId2 {
		t.Errorf("Expected character ID 2 %d, got %d", characterId2, ceremony.CharacterId2())
	}

	if ceremony.Status() != CeremonyStatusScheduled {
		t.Errorf("Expected status %s, got %s", CeremonyStatusScheduled, ceremony.Status())
	}

	if !ceremony.IsScheduled() {
		t.Error("Expected ceremony to be scheduled")
	}

	if ceremony.InviteeCount() != 0 {
		t.Errorf("Expected 0 invitees, got %d", ceremony.InviteeCount())
	}

	if !ceremony.CanStart() {
		t.Error("Expected ceremony to be startable")
	}
}

func TestCeremony_ValidationRules(t *testing.T) {
	tenantId := uuid.New()
	marriageId := uint32(1)
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	// Test zero marriage ID
	_, err := NewCeremonyBuilder(0, characterId1, characterId2, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero marriage ID")
	}
	if !contains(err.Error(), "marriage ID is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test zero character ID 1
	_, err = NewCeremonyBuilder(marriageId, 0, characterId2, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero character ID 1")
	}
	if !contains(err.Error(), "character ID 1 is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test zero character ID 2
	_, err = NewCeremonyBuilder(marriageId, characterId1, 0, tenantId).Build()
	if err == nil {
		t.Error("Expected error for zero character ID 2")
	}
	if !contains(err.Error(), "character ID 2 is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test self-ceremony validation
	_, err = NewCeremonyBuilder(marriageId, characterId1, characterId1, tenantId).Build()
	if err == nil {
		t.Error("Expected error for self-ceremony")
	}
	if !contains(err.Error(), "character cannot have ceremony with themselves") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test nil tenant ID
	_, err = NewCeremonyBuilder(marriageId, characterId1, characterId2, uuid.Nil).Build()
	if err == nil {
		t.Error("Expected error for nil tenant ID")
	}
	if !contains(err.Error(), "tenant ID is required") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test too many invitees
	tooManyInvitees := make([]uint32, MaxInvitees+1)
	for i := 0; i <= MaxInvitees; i++ {
		tooManyInvitees[i] = uint32(i + 10) // Use IDs that don't conflict with partners
	}
	_, err = NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).
		SetInvitees(tooManyInvitees).
		Build()
	if err == nil {
		t.Error("Expected error for too many invitees")
	}
	if !contains(err.Error(), "too many invitees") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test partners as invitees
	_, err = NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).
		SetInvitees([]uint32{characterId1}).
		Build()
	if err == nil {
		t.Error("Expected error for partner as invitee")
	}
	if !contains(err.Error(), "partners cannot be invitees") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}

	// Test duplicate invitees
	_, err = NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).
		SetInvitees([]uint32{uint32(10), uint32(10)}).
		Build()
	if err == nil {
		t.Error("Expected error for duplicate invitees")
	}
	if !contains(err.Error(), "duplicate invitee") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestCeremony_StateTransitionValidation(t *testing.T) {
	tenantId := uuid.New()
	marriageId := uint32(1)
	characterId1 := uint32(1)
	characterId2 := uint32(2)
	now := time.Now()

	tests := []struct {
		name        string
		status      CeremonyStatus
		startedAt   *time.Time
		completedAt *time.Time
		cancelledAt *time.Time
		postponedAt *time.Time
		expectError bool
		errorContains string
	}{
		{
			name:        "scheduled ceremony with started timestamp",
			status:      CeremonyStatusScheduled,
			startedAt:   &now,
			expectError: true,
			errorContains: "scheduled ceremony cannot have started timestamp",
		},
		{
			name:        "scheduled ceremony with completed timestamp",
			status:      CeremonyStatusScheduled,
			completedAt: &now,
			expectError: true,
			errorContains: "scheduled ceremony cannot have completed timestamp",
		},
		{
			name:        "scheduled ceremony with cancelled timestamp",
			status:      CeremonyStatusScheduled,
			cancelledAt: &now,
			expectError: true,
			errorContains: "scheduled ceremony cannot have cancelled timestamp",
		},
		{
			name:        "scheduled ceremony with postponed timestamp",
			status:      CeremonyStatusScheduled,
			postponedAt: &now,
			expectError: true,
			errorContains: "scheduled ceremony cannot have postponed timestamp",
		},
		{
			name:        "active ceremony without started timestamp",
			status:      CeremonyStatusActive,
			startedAt:   nil,
			expectError: true,
			errorContains: "active ceremony must have started timestamp",
		},
		{
			name:        "active ceremony with completed timestamp",
			status:      CeremonyStatusActive,
			startedAt:   &now,
			completedAt: &now,
			expectError: true,
			errorContains: "active ceremony cannot have completed timestamp",
		},
		{
			name:        "active ceremony with cancelled timestamp",
			status:      CeremonyStatusActive,
			startedAt:   &now,
			cancelledAt: &now,
			expectError: true,
			errorContains: "active ceremony cannot have cancelled timestamp",
		},
		{
			name:        "completed ceremony without started timestamp",
			status:      CeremonyStatusCompleted,
			startedAt:   nil,
			completedAt: &now,
			expectError: true,
			errorContains: "completed ceremony must have started timestamp",
		},
		{
			name:        "completed ceremony without completed timestamp",
			status:      CeremonyStatusCompleted,
			startedAt:   &now,
			completedAt: nil,
			expectError: true,
			errorContains: "completed ceremony must have completed timestamp",
		},
		{
			name:        "completed ceremony with cancelled timestamp",
			status:      CeremonyStatusCompleted,
			startedAt:   &now,
			completedAt: &now,
			cancelledAt: &now,
			expectError: true,
			errorContains: "completed ceremony cannot have cancelled timestamp",
		},
		{
			name:        "cancelled ceremony without cancelled timestamp",
			status:      CeremonyStatusCancelled,
			cancelledAt: nil,
			expectError: true,
			errorContains: "cancelled ceremony must have cancelled timestamp",
		},
		{
			name:        "cancelled ceremony with completed timestamp",
			status:      CeremonyStatusCancelled,
			cancelledAt: &now,
			completedAt: &now,
			expectError: true,
			errorContains: "cancelled ceremony cannot have completed timestamp",
		},
		{
			name:        "postponed ceremony without postponed timestamp",
			status:      CeremonyStatusPostponed,
			postponedAt: nil,
			expectError: true,
			errorContains: "postponed ceremony must have postponed timestamp",
		},
		{
			name:        "postponed ceremony with completed timestamp",
			status:      CeremonyStatusPostponed,
			postponedAt: &now,
			completedAt: &now,
			expectError: true,
			errorContains: "postponed ceremony cannot have completed timestamp",
		},
		{
			name:        "postponed ceremony with cancelled timestamp",
			status:      CeremonyStatusPostponed,
			postponedAt: &now,
			cancelledAt: &now,
			expectError: true,
			errorContains: "postponed ceremony cannot have cancelled timestamp",
		},
		{
			name:        "valid scheduled ceremony",
			status:      CeremonyStatusScheduled,
			expectError: false,
		},
		{
			name:        "valid active ceremony",
			status:      CeremonyStatusActive,
			startedAt:   &now,
			expectError: false,
		},
		{
			name:        "valid completed ceremony",
			status:      CeremonyStatusCompleted,
			startedAt:   &now,
			completedAt: &now,
			expectError: false,
		},
		{
			name:        "valid cancelled ceremony",
			status:      CeremonyStatusCancelled,
			cancelledAt: &now,
			expectError: false,
		},
		{
			name:        "valid postponed ceremony",
			status:      CeremonyStatusPostponed,
			postponedAt: &now,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).
				SetStatus(tt.status).
				SetStartedAt(tt.startedAt).
				SetCompletedAt(tt.completedAt).
				SetCancelledAt(tt.cancelledAt).
				SetPostponedAt(tt.postponedAt)

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s", tt.name)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestCeremony_InviteeManagement(t *testing.T) {
	tenantId := uuid.New()
	marriageId := uint32(1)
	characterId1 := uint32(1)
	characterId2 := uint32(2)
	inviteeId := uint32(10)

	ceremony, err := NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create ceremony: %v", err)
	}

	// Test adding invitee
	if !ceremony.CanAddInvitee(inviteeId) {
		t.Error("Expected to be able to add invitee")
	}

	ceremonyWithInvitee, err := ceremony.AddInvitee(inviteeId)
	if err != nil {
		t.Fatalf("Failed to add invitee: %v", err)
	}

	if ceremonyWithInvitee.InviteeCount() != 1 {
		t.Errorf("Expected 1 invitee, got %d", ceremonyWithInvitee.InviteeCount())
	}

	if !ceremonyWithInvitee.IsInvited(inviteeId) {
		t.Error("Expected invitee to be invited")
	}

	// Test cannot add duplicate invitee
	if ceremonyWithInvitee.CanAddInvitee(inviteeId) {
		t.Error("Expected to not be able to add duplicate invitee")
	}

	// Test cannot add partner as invitee
	if ceremonyWithInvitee.CanAddInvitee(characterId1) {
		t.Error("Expected to not be able to add partner as invitee")
	}

	// Test removing invitee
	if !ceremonyWithInvitee.CanRemoveInvitee(inviteeId) {
		t.Error("Expected to be able to remove invitee")
	}

	ceremonyWithoutInvitee, err := ceremonyWithInvitee.RemoveInvitee(inviteeId)
	if err != nil {
		t.Fatalf("Failed to remove invitee: %v", err)
	}

	if ceremonyWithoutInvitee.InviteeCount() != 0 {
		t.Errorf("Expected 0 invitees, got %d", ceremonyWithoutInvitee.InviteeCount())
	}

	if ceremonyWithoutInvitee.IsInvited(inviteeId) {
		t.Error("Expected invitee to not be invited")
	}

	// Test cannot remove non-existent invitee
	if ceremonyWithoutInvitee.CanRemoveInvitee(inviteeId) {
		t.Error("Expected to not be able to remove non-existent invitee")
	}

	// Test max invitees
	ceremonyWithMaxInvitees := ceremony
	for i := 0; i < MaxInvitees; i++ {
		inviteeId := uint32(i + 10)
		ceremonyWithMaxInvitees, err = ceremonyWithMaxInvitees.AddInvitee(inviteeId)
		if err != nil {
			t.Fatalf("Failed to add invitee %d: %v", inviteeId, err)
		}
	}

	if ceremonyWithMaxInvitees.InviteeCount() != MaxInvitees {
		t.Errorf("Expected %d invitees, got %d", MaxInvitees, ceremonyWithMaxInvitees.InviteeCount())
	}

	// Test cannot add more than max invitees
	if ceremonyWithMaxInvitees.CanAddInvitee(uint32(100)) {
		t.Error("Expected to not be able to add more than max invitees")
	}
}

func TestCeremony_BusinessLogic(t *testing.T) {
	tenantId := uuid.New()
	marriageId := uint32(1)
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	ceremony, err := NewCeremonyBuilder(marriageId, characterId1, characterId2, tenantId).Build()
	if err != nil {
		t.Fatalf("Failed to create ceremony: %v", err)
	}

	// Test partner identification
	if !ceremony.IsPartner(characterId1) {
		t.Error("Expected character 1 to be a partner")
	}
	if !ceremony.IsPartner(characterId2) {
		t.Error("Expected character 2 to be a partner")
	}
	if ceremony.IsPartner(uint32(3)) {
		t.Error("Expected character 3 to not be a partner")
	}

	// Test state transitions
	activeCeremony, err := ceremony.Start()
	if err != nil {
		t.Fatalf("Failed to start ceremony: %v", err)
	}
	if activeCeremony.Status() != CeremonyStatusActive {
		t.Errorf("Expected status %s, got %s", CeremonyStatusActive, activeCeremony.Status())
	}
	if !activeCeremony.IsActive() {
		t.Error("Expected ceremony to be active")
	}
	if !activeCeremony.CanComplete() {
		t.Error("Expected active ceremony to be completable")
	}

	completedCeremony, err := activeCeremony.Complete()
	if err != nil {
		t.Fatalf("Failed to complete ceremony: %v", err)
	}
	if completedCeremony.Status() != CeremonyStatusCompleted {
		t.Errorf("Expected status %s, got %s", CeremonyStatusCompleted, completedCeremony.Status())
	}
	if !completedCeremony.IsCompleted() {
		t.Error("Expected ceremony to be completed")
	}
	if !completedCeremony.IsFinished() {
		t.Error("Expected completed ceremony to be finished")
	}

	// Test cancellation
	cancelledCeremony, err := ceremony.Cancel()
	if err != nil {
		t.Fatalf("Failed to cancel ceremony: %v", err)
	}
	if cancelledCeremony.Status() != CeremonyStatusCancelled {
		t.Errorf("Expected status %s, got %s", CeremonyStatusCancelled, cancelledCeremony.Status())
	}
	if !cancelledCeremony.IsCancelled() {
		t.Error("Expected ceremony to be cancelled")
	}
	if !cancelledCeremony.IsFinished() {
		t.Error("Expected cancelled ceremony to be finished")
	}

	// Test postponement
	postponedCeremony, err := activeCeremony.Postpone()
	if err != nil {
		t.Fatalf("Failed to postpone ceremony: %v", err)
	}
	if postponedCeremony.Status() != CeremonyStatusPostponed {
		t.Errorf("Expected status %s, got %s", CeremonyStatusPostponed, postponedCeremony.Status())
	}
	if !postponedCeremony.IsPostponed() {
		t.Error("Expected ceremony to be postponed")
	}
	if !postponedCeremony.CanStart() {
		t.Error("Expected postponed ceremony to be startable")
	}

	// Test rescheduling - start with a fresh ceremony to avoid timing issues
	newScheduledAt := time.Now().Add(24 * time.Hour)
	rescheduledCeremony, err := postponedCeremony.Reschedule(newScheduledAt)
	if err != nil {
		t.Fatalf("Failed to reschedule ceremony: %v", err)
	}
	if rescheduledCeremony.Status() != CeremonyStatusScheduled {
		t.Errorf("Expected status %s, got %s", CeremonyStatusScheduled, rescheduledCeremony.Status())
	}
	if !rescheduledCeremony.IsScheduled() {
		t.Error("Expected ceremony to be scheduled")
	}
	if !rescheduledCeremony.CanReschedule() {
		t.Error("Expected rescheduled ceremony to be reschedulable")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}