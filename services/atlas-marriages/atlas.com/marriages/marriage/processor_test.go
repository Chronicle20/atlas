package marriage

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// setupTestContext creates a context with tenant information
func setupTestContext(tenantId uuid.UUID) context.Context {
	ctx := context.Background()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		panic(err)
	}
	return tenant.WithContext(ctx, tenantModel)
}

// setupTestData creates test data in the database
func setupTestData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	// Create some test marriages and proposals
	now := time.Now()
	engaged := now
	married := now
	
	// Create an active marriage for character 100
	activeMarriage := Entity{
		ID:           1,
		CharacterId1: 100,
		CharacterId2: 101,
		Status:       StatusMarried,
		ProposedAt:   now.Add(-48 * time.Hour),
		EngagedAt:    &engaged,
		MarriedAt:    &married,
		TenantId:     tenantId,
		CreatedAt:    now.Add(-48 * time.Hour),
		UpdatedAt:    now,
	}
	if err := db.Create(&activeMarriage).Error; err != nil {
		t.Fatalf("Failed to create test marriage: %v", err)
	}

	// Create a recent proposal for global cooldown testing
	recentProposalTime := now.Add(-2 * time.Hour)
	recentCooldownUntil := now.Add(22 * time.Hour)
	recentProposal := ProposalEntity{
		ID:           1,
		ProposerId:   200,
		TargetId:     201,
		Status:       ProposalStatusRejected,
		ProposedAt:   recentProposalTime, // 2 hours ago (within 4-hour global cooldown)
		RespondedAt:  &recentProposalTime,
		ExpiresAt:    recentProposalTime.Add(ProposalExpiryDuration),
		RejectionCount: 1,
		CooldownUntil: &recentCooldownUntil, // Must have cooldown for rejected proposals
		TenantId:     tenantId,
		CreatedAt:    recentProposalTime,
		UpdatedAt:    recentProposalTime,
	}
	if err := db.Create(&recentProposal).Error; err != nil {
		t.Fatalf("Failed to create test proposal: %v", err)
	}

	// Create a rejected proposal for per-target cooldown testing
	rejectedProposalTime := now.Add(-12 * time.Hour)
	cooldownUntil := now.Add(12 * time.Hour)
	rejectedProposal := ProposalEntity{
		ID:           2,
		ProposerId:   300,
		TargetId:     301,
		Status:       ProposalStatusRejected,
		ProposedAt:   rejectedProposalTime, // 12 hours ago
		RespondedAt:  &rejectedProposalTime,
		ExpiresAt:    rejectedProposalTime.Add(ProposalExpiryDuration),
		RejectionCount: 1,
		CooldownUntil: &cooldownUntil, // 12 hours from now (within cooldown)
		TenantId:     tenantId,
		CreatedAt:    rejectedProposalTime,
		UpdatedAt:    rejectedProposalTime,
	}
	if err := db.Create(&rejectedProposal).Error; err != nil {
		t.Fatalf("Failed to create test rejected proposal: %v", err)
	}

	// Create an expired proposal for per-target cooldown testing
	expiredProposalTime := now.Add(-26 * time.Hour)
	expiredProposal := ProposalEntity{
		ID:           3,
		ProposerId:   400,
		TargetId:     401,
		Status:       ProposalStatusExpired,
		ProposedAt:   expiredProposalTime, // 26 hours ago
		ExpiresAt:    expiredProposalTime.Add(ProposalExpiryDuration), // Expired 2 hours ago
		RejectionCount: 0,
		TenantId:     tenantId,
		CreatedAt:    expiredProposalTime,
		UpdatedAt:    now.Add(-2 * time.Hour), // Updated when expired
	}
	if err := db.Create(&expiredProposal).Error; err != nil {
		t.Fatalf("Failed to create test expired proposal: %v", err)
	}

	// Create a pending proposal for active proposal testing
	pendingProposalTime := now.Add(-1 * time.Hour)
	pendingProposal := ProposalEntity{
		ID:           4,
		ProposerId:   500,
		TargetId:     501,
		Status:       ProposalStatusPending,
		ProposedAt:   pendingProposalTime, // 1 hour ago
		ExpiresAt:    pendingProposalTime.Add(ProposalExpiryDuration), // Expires in 23 hours
		RejectionCount: 0,
		TenantId:     tenantId,
		CreatedAt:    pendingProposalTime,
		UpdatedAt:    pendingProposalTime,
	}
	if err := db.Create(&pendingProposal).Error; err != nil {
		t.Fatalf("Failed to create test pending proposal: %v", err)
	}
}

func TestProcessor_CheckEligibility(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db)

	// Test basic eligibility check
	eligible, err := processor.CheckEligibility(1)()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !eligible {
		t.Error("Expected character to be eligible")
	}
}

func TestProcessor_CheckProposalEligibility(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name        string
		proposerId  uint32
		targetId    uint32
		expected    bool
		description string
	}{
		{
			name:        "valid proposal between unattached characters",
			proposerId:  1,
			targetId:    2,
			expected:    true,
			description: "Both characters are free and eligible",
		},
		{
			name:        "proposer already married",
			proposerId:  100, // This character is married in test data
			targetId:    2,
			expected:    false,
			description: "Proposer is already married",
		},
		{
			name:        "target already married",
			proposerId:  1,
			targetId:    101, // This character is married in test data
			expected:    false,
			description: "Target is already married",
		},
		{
			name:        "active proposal already exists",
			proposerId:  500, // This character has pending proposal in test data
			targetId:    501,
			expected:    false,
			description: "Active proposal already exists between these characters",
		},
		{
			name:        "self proposal",
			proposerId:  1,
			targetId:    1,
			expected:    true, // CheckProposalEligibility doesn't validate self-proposal, that's in the model
			description: "Self proposal should be allowed by eligibility check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, err := processor.CheckProposalEligibility(tt.proposerId, tt.targetId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if eligible != tt.expected {
				t.Errorf("Expected eligibility %v, got %v for %s", tt.expected, eligible, tt.description)
			}
		})
	}
}

func TestProcessor_CheckGlobalCooldown(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name        string
		proposerId  uint32
		expected    bool
		description string
	}{
		{
			name:        "no previous proposals",
			proposerId:  1,
			expected:    true,
			description: "Character with no previous proposals should be able to propose",
		},
		{
			name:        "within global cooldown",
			proposerId:  200, // This character has a proposal 2 hours ago (within 4-hour cooldown)
			expected:    false,
			description: "Character should be in global cooldown",
		},
		{
			name:        "outside global cooldown",
			proposerId:  400, // This character has a proposal 26 hours ago (outside 4-hour cooldown)
			expected:    true,
			description: "Character should be outside global cooldown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canPropose, err := processor.CheckGlobalCooldown(tt.proposerId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if canPropose != tt.expected {
				t.Errorf("Expected global cooldown result %v, got %v for %s", tt.expected, canPropose, tt.description)
			}
		})
	}
}

func TestProcessor_CheckPerTargetCooldown(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name        string
		proposerId  uint32
		targetId    uint32
		expected    bool
		description string
	}{
		{
			name:        "no previous proposals to target",
			proposerId:  1,
			targetId:    2,
			expected:    true,
			description: "Character with no previous proposals to target should be able to propose",
		},
		{
			name:        "within per-target cooldown after rejection",
			proposerId:  300, // This character has rejected proposal to 301 with cooldown
			targetId:    301,
			expected:    false,
			description: "Character should be in per-target cooldown after rejection",
		},
		{
			name:        "within per-target cooldown after expiry",
			proposerId:  400, // This character has expired proposal to 401
			targetId:    401,
			expected:    false,
			description: "Character should be in per-target cooldown after expiry",
		},
		{
			name:        "can propose to different target",
			proposerId:  300, // This character has cooldown with 301, but not 302
			targetId:    302,
			expected:    true,
			description: "Character should be able to propose to different target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canPropose, err := processor.CheckPerTargetCooldown(tt.proposerId, tt.targetId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if canPropose != tt.expected {
				t.Errorf("Expected per-target cooldown result %v, got %v for %s", tt.expected, canPropose, tt.description)
			}
		})
	}
}

func TestProcessor_GetActiveProposal(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name        string
		proposerId  uint32
		targetId    uint32
		expectFound bool
		description string
	}{
		{
			name:        "no active proposal",
			proposerId:  1,
			targetId:    2,
			expectFound: false,
			description: "No active proposal should exist",
		},
		{
			name:        "active proposal exists",
			proposerId:  500, // This character has pending proposal to 501
			targetId:    501,
			expectFound: true,
			description: "Active proposal should be found",
		},
		{
			name:        "rejected proposal not considered active",
			proposerId:  200, // This character has rejected proposal to 201
			targetId:    201,
			expectFound: false,
			description: "Rejected proposal should not be considered active",
		},
		{
			name:        "expired proposal not considered active",
			proposerId:  400, // This character has expired proposal to 401
			targetId:    401,
			expectFound: false,
			description: "Expired proposal should not be considered active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, err := processor.GetActiveProposal(tt.proposerId, tt.targetId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			found := proposal != nil
			if found != tt.expectFound {
				t.Errorf("Expected active proposal found %v, got %v for %s", tt.expectFound, found, tt.description)
			}
			
			if found {
				if proposal.ProposerId() != tt.proposerId {
					t.Errorf("Expected proposer ID %d, got %d", tt.proposerId, proposal.ProposerId())
				}
				if proposal.TargetId() != tt.targetId {
					t.Errorf("Expected target ID %d, got %d", tt.targetId, proposal.TargetId())
				}
				if !proposal.IsPending() {
					t.Error("Expected active proposal to be pending")
				}
			}
		})
	}
}

func TestProcessor_GetPendingProposalsByCharacter(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name         string
		characterId  uint32
		expectedCount int
		description  string
	}{
		{
			name:         "no pending proposals",
			characterId:  1,
			expectedCount: 0,
			description:  "Character with no pending proposals",
		},
		{
			name:         "has pending proposal as proposer",
			characterId:  500, // This character has pending proposal to 501
			expectedCount: 1,
			description:  "Character with pending proposal as proposer",
		},
		{
			name:         "has pending proposal as target",
			characterId:  501, // This character has pending proposal from 500
			expectedCount: 1,
			description:  "Character with pending proposal as target",
		},
		{
			name:         "has only non-pending proposals",
			characterId:  200, // This character has rejected proposal
			expectedCount: 0,
			description:  "Character with only non-pending proposals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposals, err := processor.GetPendingProposalsByCharacter(tt.characterId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if len(proposals) != tt.expectedCount {
				t.Errorf("Expected %d pending proposals, got %d for %s", tt.expectedCount, len(proposals), tt.description)
			}
			
			// Verify all returned proposals are actually pending
			for _, proposal := range proposals {
				if !proposal.IsPending() {
					t.Error("Expected all returned proposals to be pending")
				}
				if proposal.TenantId() != tenantId {
					t.Error("Expected all returned proposals to have correct tenant ID")
				}
			}
		})
	}
}

func TestProcessor_GetProposalHistory(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name         string
		proposerId   uint32
		targetId     uint32
		expectedCount int
		description  string
	}{
		{
			name:         "no proposal history",
			proposerId:   1,
			targetId:     2,
			expectedCount: 0,
			description:  "Characters with no proposal history",
		},
		{
			name:         "has proposal history",
			proposerId:   200, // This character has proposal to 201
			targetId:     201,
			expectedCount: 1,
			description:  "Characters with proposal history",
		},
		{
			name:         "multiple proposal history",
			proposerId:   500, // This character has proposal to 501
			targetId:     501,
			expectedCount: 1,
			description:  "Characters with multiple proposal history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := processor.GetProposalHistory(tt.proposerId, tt.targetId)()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if len(history) != tt.expectedCount {
				t.Errorf("Expected %d proposal history entries, got %d for %s", tt.expectedCount, len(history), tt.description)
			}
			
			// Verify all returned proposals have correct proposer and target
			for _, proposal := range history {
				if proposal.ProposerId() != tt.proposerId {
					t.Errorf("Expected proposer ID %d, got %d", tt.proposerId, proposal.ProposerId())
				}
				if proposal.TargetId() != tt.targetId {
					t.Errorf("Expected target ID %d, got %d", tt.targetId, proposal.TargetId())
				}
				if proposal.TenantId() != tenantId {
					t.Error("Expected all returned proposals to have correct tenant ID")
				}
			}
		})
	}
}

func TestProcessor_Propose_Success(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db)

	proposerId := uint32(1)
	targetId := uint32(2)

	proposal, err := processor.Propose(proposerId, targetId)()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if proposal.ProposerId() != proposerId {
		t.Errorf("Expected proposer ID %d, got %d", proposerId, proposal.ProposerId())
	}

	if proposal.TargetId() != targetId {
		t.Errorf("Expected target ID %d, got %d", targetId, proposal.TargetId())
	}

	if !proposal.IsPending() {
		t.Error("Expected proposal to be pending")
	}

	if proposal.TenantId() != tenantId {
		t.Error("Expected proposal to have correct tenant ID")
	}

	if proposal.RejectionCount() != 0 {
		t.Error("Expected new proposal to have zero rejection count")
	}

	if proposal.IsExpired() {
		t.Error("Expected new proposal to not be expired")
	}

	// Verify the proposal was saved to database
	var savedProposal ProposalEntity
	err = db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ?", proposerId, targetId, tenantId).First(&savedProposal).Error
	if err != nil {
		t.Fatalf("Failed to retrieve saved proposal: %v", err)
	}

	if savedProposal.Status != ProposalStatusPending {
		t.Error("Expected saved proposal to be pending")
	}
}

func TestProcessor_Propose_EligibilityFailures(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	processor := NewProcessor(log, ctx, db)

	tests := []struct {
		name        string
		proposerId  uint32
		targetId    uint32
		expectError bool
		description string
	}{
		{
			name:        "proposer already married",
			proposerId:  100, // This character is married in test data
			targetId:    2,
			expectError: true,
			description: "Should fail when proposer is already married",
		},
		{
			name:        "target already married",
			proposerId:  1,
			targetId:    101, // This character is married in test data
			expectError: true,
			description: "Should fail when target is already married",
		},
		{
			name:        "active proposal exists",
			proposerId:  500, // This character has pending proposal in test data
			targetId:    501,
			expectError: true,
			description: "Should fail when active proposal already exists",
		},
		{
			name:        "global cooldown active",
			proposerId:  200, // This character has proposal 2 hours ago (within 4-hour cooldown)
			targetId:    2,
			expectError: true,
			description: "Should fail when proposer is in global cooldown",
		},
		{
			name:        "per-target cooldown active",
			proposerId:  300, // This character has rejected proposal to 301 with cooldown
			targetId:    301,
			expectError: true,
			description: "Should fail when proposer is in per-target cooldown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processor.Propose(tt.proposerId, tt.targetId)()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}
		})
	}
}

func TestProcessor_ProposeAndEmit(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db)

	proposerId := uint32(1)
	targetId := uint32(2)
	transactionId := uuid.New()

	// Test successful proposal with emit
	proposal, err := processor.ProposeAndEmit(transactionId, proposerId, targetId)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if proposal.ProposerId() != proposerId {
		t.Errorf("Expected proposer ID %d, got %d", proposerId, proposal.ProposerId())
	}

	if proposal.TargetId() != targetId {
		t.Errorf("Expected target ID %d, got %d", targetId, proposal.TargetId())
	}

	if !proposal.IsPending() {
		t.Error("Expected proposal to be pending")
	}

	// Test failure case
	_, err = processor.ProposeAndEmit(transactionId, proposerId, targetId)
	if err == nil {
		t.Error("Expected error when proposing to same target twice")
	}
}

func TestProcessor_EligibilityConstants(t *testing.T) {
	// Test that the eligibility requirement constant is properly defined
	if EligibilityRequirement != 10 {
		t.Errorf("Expected eligibility requirement to be 10, got %d", EligibilityRequirement)
	}
}

func TestProcessor_ContextTenantExtraction(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	log := logrus.New()
	
	// Test with context that has tenant
	ctx := setupTestContext(tenantId)
	processor := NewProcessor(log, ctx, db)
	
	// This should work without error
	_, err := processor.CheckEligibility(1)()
	if err != nil {
		t.Fatalf("Unexpected error with valid tenant context: %v", err)
	}
	
	// Test with context that doesn't have tenant - this should panic
	emptyCtx := context.Background()
	processorWithoutTenant := NewProcessor(log, emptyCtx, db)
	
	// This should panic when trying to extract tenant
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when tenant is not in context")
		}
	}()
	
	_, _ = processorWithoutTenant.CheckProposalEligibility(1, 2)()
}

func TestProcessor_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db)

	// Test with invalid database connection - close the database
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get SQL DB: %v", err)
	}
	sqlDB.Close()

	// This should return an error due to closed database
	_, err = processor.CheckProposalEligibility(1, 2)()
	if err == nil {
		t.Error("Expected error when database is closed")
	}
}

func TestProcessor_ProducerFunctionPlaceholder(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db).(*ProcessorImpl)

	// Test that producer function is nil (placeholder)
	if processor.producer != nil {
		t.Error("Expected producer function to be nil (placeholder)")
	}
}

func TestProcessor_EligibilityErrors(t *testing.T) {
	// Test the predefined eligibility errors
	tests := []struct {
		name     string
		err      EligibilityError
		expected string
	}{
		{
			name:     "character too low level",
			err:      ErrCharacterTooLowLevel,
			expected: "character level is too low for marriage",
		},
		{
			name:     "character already married",
			err:      ErrCharacterAlreadyMarried,
			expected: "character is already married or engaged",
		},
		{
			name:     "target already engaged",
			err:      ErrTargetAlreadyEngaged,
			expected: "target character already has a pending proposal",
		},
		{
			name:     "global cooldown active",
			err:      ErrGlobalCooldownActive,
			expected: "proposer is in global cooldown period",
		},
		{
			name:     "target cooldown active",
			err:      ErrTargetCooldownActive,
			expected: "proposer is in cooldown period for this target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestProcessor_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	// Test concurrent access to the same processor instance
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	// Start two goroutines that try to create proposals simultaneously
	go func() {
		processor := NewProcessor(log, ctx, db)
		_, err := processor.Propose(1, 2)()
		errors <- err
		done <- true
	}()

	go func() {
		processor := NewProcessor(log, ctx, db)
		_, err := processor.Propose(3, 4)()
		errors <- err
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Check that both succeeded or at least one succeeded
	err1 := <-errors
	err2 := <-errors

	if err1 != nil && err2 != nil {
		t.Errorf("Both concurrent proposals failed: %v, %v", err1, err2)
	}
}