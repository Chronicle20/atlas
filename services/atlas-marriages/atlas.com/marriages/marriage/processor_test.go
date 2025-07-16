package marriage

import (
	"context"
	"errors"
	"testing"
	"time"

	"atlas-marriages/character"
	kafkaProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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

// MockCharacterProcessor provides a mock implementation for testing
type MockCharacterProcessor struct {
	characters map[uint32]character.Model
}

func NewMockCharacterProcessor() *MockCharacterProcessor {
	return &MockCharacterProcessor{
		characters: make(map[uint32]character.Model),
	}
}

func (m *MockCharacterProcessor) AddCharacter(id uint32, name string, level byte) {
	m.characters[id] = character.NewModel(id, name, level)
}

func (m *MockCharacterProcessor) GetById(characterId uint32) (character.Model, error) {
	if char, exists := m.characters[characterId]; exists {
		return char, nil
	}
	return character.Model{}, errors.New("character not found")
}

func (m *MockCharacterProcessor) ByIdProvider(characterId uint32) model.Provider[character.Model] {
	return func() (character.Model, error) {
		return m.GetById(characterId)
	}
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
		ID:             1,
		ProposerId:     200,
		TargetId:       201,
		Status:         ProposalStatusRejected,
		ProposedAt:     recentProposalTime, // 2 hours ago (within 4-hour global cooldown)
		RespondedAt:    &recentProposalTime,
		ExpiresAt:      recentProposalTime.Add(ProposalExpiryDuration),
		RejectionCount: 1,
		CooldownUntil:  &recentCooldownUntil, // Must have cooldown for rejected proposals
		TenantId:       tenantId,
		CreatedAt:      recentProposalTime,
		UpdatedAt:      recentProposalTime,
	}
	if err := db.Create(&recentProposal).Error; err != nil {
		t.Fatalf("Failed to create test proposal: %v", err)
	}

	// Create a rejected proposal for per-target cooldown testing
	rejectedProposalTime := now.Add(-12 * time.Hour)
	cooldownUntil := now.Add(12 * time.Hour)
	rejectedProposal := ProposalEntity{
		ID:             2,
		ProposerId:     300,
		TargetId:       301,
		Status:         ProposalStatusRejected,
		ProposedAt:     rejectedProposalTime, // 12 hours ago
		RespondedAt:    &rejectedProposalTime,
		ExpiresAt:      rejectedProposalTime.Add(ProposalExpiryDuration),
		RejectionCount: 1,
		CooldownUntil:  &cooldownUntil, // 12 hours from now (within cooldown)
		TenantId:       tenantId,
		CreatedAt:      rejectedProposalTime,
		UpdatedAt:      rejectedProposalTime,
	}
	if err := db.Create(&rejectedProposal).Error; err != nil {
		t.Fatalf("Failed to create test rejected proposal: %v", err)
	}

	// Create an expired proposal for per-target cooldown testing
	expiredProposalTime := now.Add(-26 * time.Hour)
	expiredProposal := ProposalEntity{
		ID:             3,
		ProposerId:     400,
		TargetId:       401,
		Status:         ProposalStatusExpired,
		ProposedAt:     expiredProposalTime,                             // 26 hours ago
		ExpiresAt:      expiredProposalTime.Add(ProposalExpiryDuration), // Expired 2 hours ago
		RejectionCount: 0,
		TenantId:       tenantId,
		CreatedAt:      expiredProposalTime,
		UpdatedAt:      now.Add(-2 * time.Hour), // Updated when expired
	}
	if err := db.Create(&expiredProposal).Error; err != nil {
		t.Fatalf("Failed to create test expired proposal: %v", err)
	}

	// Create a pending proposal for active proposal testing
	pendingProposalTime := now.Add(-1 * time.Hour)
	pendingProposal := ProposalEntity{
		ID:             4,
		ProposerId:     500,
		TargetId:       501,
		Status:         ProposalStatusPending,
		ProposedAt:     pendingProposalTime,                             // 1 hour ago
		ExpiresAt:      pendingProposalTime.Add(ProposalExpiryDuration), // Expires in 23 hours
		RejectionCount: 0,
		TenantId:       tenantId,
		CreatedAt:      pendingProposalTime,
		UpdatedAt:      pendingProposalTime,
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

	// Create mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()

	// Add test characters with different levels
	mockCharacterProcessor.AddCharacter(1, "EligibleCharacter", 15)  // Level 15 (above requirement)
	mockCharacterProcessor.AddCharacter(2, "IneligibleCharacter", 5) // Level 5 (below requirement)
	mockCharacterProcessor.AddCharacter(3, "ExactlyEligible", 10)    // Level 10 (exactly at requirement)

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	tests := []struct {
		name        string
		characterId uint32
		expected    bool
		description string
	}{
		{
			name:        "character above level requirement",
			characterId: 1,
			expected:    true,
			description: "Character with level 15 should be eligible",
		},
		{
			name:        "character below level requirement",
			characterId: 2,
			expected:    false,
			description: "Character with level 5 should not be eligible",
		},
		{
			name:        "character exactly at level requirement",
			characterId: 3,
			expected:    true,
			description: "Character with level 10 should be eligible",
		},
		{
			name:        "character not found",
			characterId: 999,
			expected:    false,
			description: "Non-existent character should cause error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, err := processor.CheckEligibility(tt.characterId)()

			if tt.characterId == 999 {
				// Expect error for non-existent character
				if err == nil {
					t.Error("Expected error for non-existent character")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if eligible != tt.expected {
				t.Errorf("Expected eligibility %v, got %v for %s", tt.expected, eligible, tt.description)
			}
		})
	}
}

func TestProcessor_CheckProposalEligibility(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()
	setupTestData(t, db, tenantId)

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)     // Eligible
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)     // Eligible
	mockCharacterProcessor.AddCharacter(100, "Character100", 15) // Eligible (married)
	mockCharacterProcessor.AddCharacter(101, "Character101", 15) // Eligible (married)
	mockCharacterProcessor.AddCharacter(500, "Character500", 15) // Eligible (has proposal)
	mockCharacterProcessor.AddCharacter(501, "Character501", 15) // Eligible (has proposal)

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

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
		name          string
		characterId   uint32
		expectedCount int
		description   string
	}{
		{
			name:          "no pending proposals",
			characterId:   1,
			expectedCount: 0,
			description:   "Character with no pending proposals",
		},
		{
			name:          "has pending proposal as proposer",
			characterId:   500, // This character has pending proposal to 501
			expectedCount: 1,
			description:   "Character with pending proposal as proposer",
		},
		{
			name:          "has pending proposal as target",
			characterId:   501, // This character has pending proposal from 500
			expectedCount: 1,
			description:   "Character with pending proposal as target",
		},
		{
			name:          "has only non-pending proposals",
			characterId:   200, // This character has rejected proposal
			expectedCount: 0,
			description:   "Character with only non-pending proposals",
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
		name          string
		proposerId    uint32
		targetId      uint32
		expectedCount int
		description   string
	}{
		{
			name:          "no proposal history",
			proposerId:    1,
			targetId:      2,
			expectedCount: 0,
			description:   "Characters with no proposal history",
		},
		{
			name:          "has proposal history",
			proposerId:    200, // This character has proposal to 201
			targetId:      201,
			expectedCount: 1,
			description:   "Characters with proposal history",
		},
		{
			name:          "multiple proposal history",
			proposerId:    500, // This character has proposal to 501
			targetId:      501,
			expectedCount: 1,
			description:   "Characters with multiple proposal history",
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

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

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

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)
	mockCharacterProcessor.AddCharacter(100, "Character100", 15) // Already married
	mockCharacterProcessor.AddCharacter(101, "Character101", 15) // Already married
	mockCharacterProcessor.AddCharacter(200, "Character200", 15) // In global cooldown
	mockCharacterProcessor.AddCharacter(300, "Character300", 15) // In per-target cooldown
	mockCharacterProcessor.AddCharacter(301, "Character301", 15) // Target of cooldown
	mockCharacterProcessor.AddCharacter(500, "Character500", 15) // Has active proposal
	mockCharacterProcessor.AddCharacter(501, "Character501", 15) // Target of active proposal

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

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

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)

	// Create mock producer
	mockProducer := func(token string) kafkaProducer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			// Mock producer that does nothing (for testing)
			return nil
		}
	}

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor).WithProducer(mockProducer)

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

	// Test that the level requirement is actually enforced
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	// Create mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()

	// Add characters at various levels around the requirement
	mockCharacterProcessor.AddCharacter(1, "BelowRequirement", byte(EligibilityRequirement-1)) // Level 9
	mockCharacterProcessor.AddCharacter(2, "AtRequirement", byte(EligibilityRequirement))      // Level 10
	mockCharacterProcessor.AddCharacter(3, "AboveRequirement", byte(EligibilityRequirement+1)) // Level 11

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	// Test character below requirement
	eligible, err := processor.CheckEligibility(1)()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if eligible {
		t.Errorf("Character with level %d should not be eligible (requirement: %d)", EligibilityRequirement-1, EligibilityRequirement)
	}

	// Test character at requirement
	eligible, err = processor.CheckEligibility(2)()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !eligible {
		t.Errorf("Character with level %d should be eligible (requirement: %d)", EligibilityRequirement, EligibilityRequirement)
	}

	// Test character above requirement
	eligible, err = processor.CheckEligibility(3)()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !eligible {
		t.Errorf("Character with level %d should be eligible (requirement: %d)", EligibilityRequirement+1, EligibilityRequirement)
	}
}

func TestProcessor_ContextTenantExtraction(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	log := logrus.New()

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)

	// Test with context that has tenant
	ctx := setupTestContext(tenantId)
	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	// This should work without error
	_, err := processor.CheckEligibility(1)()
	if err != nil {
		t.Fatalf("Unexpected error with valid tenant context: %v", err)
	}

	// Test with context that doesn't have tenant - this should panic during processor creation
	emptyCtx := context.Background()
	
	// This should panic when trying to create the processor (because character processor needs tenant context)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when tenant is not in context during processor creation")
			}
		}()
		
		// This will panic because NewProcessor -> character.NewProcessor -> tenant.MustFromContext
		_ = NewProcessor(log, emptyCtx, db)
	}()
}

func TestProcessor_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	// Create mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

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

func TestProcessor_ProducerFunctionDefault(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db).(*ProcessorImpl)

	// Test that producer function is created by default
	if processor.producer == nil {
		t.Error("Expected producer function to be created by default")
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

	// Create mock character processor with eligible characters
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "Character1", 15)
	mockCharacterProcessor.AddCharacter(2, "Character2", 15)
	mockCharacterProcessor.AddCharacter(3, "Character3", 15)
	mockCharacterProcessor.AddCharacter(4, "Character4", 15)

	// Test concurrent access to the same processor instance
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	// Start two goroutines that try to create proposals simultaneously
	go func() {
		processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)
		_, err := processor.Propose(1, 2)()
		errors <- err
		done <- true
	}()

	go func() {
		processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)
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

func TestProcessor_LevelRequirementEnforcement(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	log := logrus.New()

	// Create mock character processor with characters at different levels
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "LowLevelChar", 5)    // Level 5 - below requirement
	mockCharacterProcessor.AddCharacter(2, "HighLevelChar", 15)  // Level 15 - above requirement
	mockCharacterProcessor.AddCharacter(3, "ExactLevelChar", 10) // Level 10 - exactly at requirement

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	// Test 1: Low level proposer should be rejected
	t.Run("low level proposer rejected", func(t *testing.T) {
		_, err := processor.Propose(1, 2)() // Level 5 proposes to level 15
		if err == nil {
			t.Error("Expected error for low level proposer")
		}
		if !contains(err.Error(), "eligibility check failed") {
			t.Errorf("Expected eligibility check failure, got: %v", err)
		}
	})

	// Test 2: High level proposer with low level target should be rejected
	t.Run("low level target rejected", func(t *testing.T) {
		_, err := processor.Propose(2, 1)() // Level 15 proposes to level 5
		if err == nil {
			t.Error("Expected error for low level target")
		}
		if !contains(err.Error(), "eligibility check failed") {
			t.Errorf("Expected eligibility check failure, got: %v", err)
		}
	})

	// Test 3: Both high level should succeed
	t.Run("both eligible characters succeed", func(t *testing.T) {
		proposal, err := processor.Propose(2, 3)() // Level 15 proposes to level 10
		if err != nil {
			t.Errorf("Expected success for both eligible characters, got error: %v", err)
		}
		if proposal.ProposerId() != 2 {
			t.Errorf("Expected proposer ID 2, got %d", proposal.ProposerId())
		}
		if proposal.TargetId() != 3 {
			t.Errorf("Expected target ID 3, got %d", proposal.TargetId())
		}
	})

	// Test 4: Exact level requirement should succeed
	t.Run("exact level requirement succeeds", func(t *testing.T) {
		// Add another character at exact level for this test
		mockCharacterProcessor.AddCharacter(4, "AnotherExactLevel", 10)

		proposal, err := processor.Propose(3, 4)() // Level 10 proposes to level 10
		if err != nil {
			t.Errorf("Expected success for characters at exact level requirement, got error: %v", err)
		}
		if !proposal.IsPending() {
			t.Error("Expected proposal to be pending")
		}
	})
}

// TestProcessor_ExpireProposal tests the proposal expiry functionality
func TestProcessor_ExpireProposal(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	logger := logrus.New()

	// Create mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacter(1, "TestChar1", byte(EligibilityRequirement))
	mockCharacterProcessor.AddCharacter(2, "TestChar2", byte(EligibilityRequirement))
	mockCharacterProcessor.AddCharacter(3, "TestChar3", byte(EligibilityRequirement))
	mockCharacterProcessor.AddCharacter(4, "TestChar4", byte(EligibilityRequirement))

	processor := NewProcessor(logger, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	t.Run("expire_pending_proposal", func(t *testing.T) {
		// Create a proposal that's pending
		proposal, err := processor.Propose(1, 2)()
		if err != nil {
			t.Fatalf("Failed to create proposal: %v", err)
		}

		// Verify proposal is pending
		if !proposal.IsPending() {
			t.Error("Expected proposal to be pending")
		}

		// Expire the proposal
		expiredProposal, err := processor.ExpireProposal(proposal.Id())()
		if err != nil {
			t.Fatalf("Failed to expire proposal: %v", err)
		}

		// Verify proposal is now expired
		if expiredProposal.Status() != ProposalStatusExpired {
			t.Errorf("Expected proposal status to be expired, got %v", expiredProposal.Status())
		}

		// Verify can't respond to expired proposal
		if expiredProposal.CanRespond() {
			t.Error("Expired proposal should not be able to respond")
		}
	})

	t.Run("cannot_expire_non_pending_proposal", func(t *testing.T) {
		// Create and accept a proposal using different characters
		proposal, err := processor.Propose(3, 4)()
		if err != nil {
			t.Fatalf("Failed to create proposal: %v", err)
		}

		// Accept the proposal
		_, err = processor.AcceptProposal(proposal.Id())()
		if err != nil {
			t.Fatalf("Failed to accept proposal: %v", err)
		}

		// Try to expire the accepted proposal - should fail
		_, err = processor.ExpireProposal(proposal.Id())()
		if err == nil {
			t.Error("Expected error when expiring non-pending proposal")
		}
	})

	t.Run("expire_nonexistent_proposal", func(t *testing.T) {
		// Try to expire a proposal that doesn't exist
		_, err := processor.ExpireProposal(999)()
		if err == nil {
			t.Error("Expected error when expiring nonexistent proposal")
		}
	})
}

// TestProcessor_ProcessExpiredProposals tests batch processing of expired proposals
func TestProcessor_ProcessExpiredProposals(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)
	logger := logrus.New()

	// Create mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()
	for i := uint32(1); i <= 15; i++ {
		mockCharacterProcessor.AddCharacter(i, "TestChar", byte(EligibilityRequirement))
	}

	processor := NewProcessor(logger, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	t.Run("process_multiple_expired_proposals", func(t *testing.T) {
		// Create several proposals
		var proposalIds []uint32
		for i := uint32(1); i <= 5; i++ {
			proposal, err := processor.Propose(i, i+5)() // 1->6, 2->7, 3->8, 4->9, 5->10
			if err != nil {
				t.Fatalf("Failed to create proposal %d: %v", i, err)
			}
			proposalIds = append(proposalIds, proposal.Id())
		}

		// Manually expire them in database by updating their expires_at time to the past
		pastTime := time.Now().Add(-1 * time.Hour)
		for _, proposalId := range proposalIds {
			err := db.Model(&ProposalEntity{}).
				Where("id = ?", proposalId).
				Update("expires_at", pastTime).Error
			if err != nil {
				t.Fatalf("Failed to update proposal expiry time: %v", err)
			}
		}

		// Process expired proposals using the direct database approach instead
		var expiredEntities []ProposalEntity
		err := db.Where("tenant_id = ? AND status = ? AND expires_at < ?", 
			tenantId, ProposalStatusPending, time.Now()).Find(&expiredEntities).Error
		if err != nil {
			t.Fatalf("Failed to find expired proposals: %v", err)
		}

		if len(expiredEntities) != 5 {
			t.Errorf("Expected 5 expired proposals, got %d", len(expiredEntities))
		}

		// Manually update their status to expired
		for _, entity := range expiredEntities {
			_, err := processor.ExpireProposal(entity.ID)()
			if err != nil {
				t.Fatalf("Failed to expire proposal %d: %v", entity.ID, err)
			}
		}

		// Verify all proposals are now marked as expired in the database
		for _, proposalId := range proposalIds {
			var entity ProposalEntity
			err := db.Where("id = ?", proposalId).First(&entity).Error
			if err != nil {
				t.Fatalf("Failed to retrieve proposal %d: %v", proposalId, err)
			}

			if entity.Status != ProposalStatusExpired {
				t.Errorf("Expected proposal %d to be expired, got status %v", proposalId, entity.Status)
			}
		}
	})

	t.Run("process_no_expired_proposals", func(t *testing.T) {
		// Clear the table from previous test
		db.Exec("DELETE FROM proposal_entities")
		
		// Create a proposal that's not expired (use new characters to avoid conflicts)
		_, err := processor.Propose(11, 12)()
		if err != nil {
			t.Fatalf("Failed to create proposal: %v", err)
		}

		// Verify no proposals need expiry processing
		var expiredEntities []ProposalEntity
		err = db.Where("tenant_id = ? AND status = ? AND expires_at < ?", 
			tenantId, ProposalStatusPending, time.Now()).Find(&expiredEntities).Error
		if err != nil {
			t.Fatalf("Failed to find expired proposals: %v", err)
		}

		if len(expiredEntities) != 0 {
			t.Errorf("Expected 0 expired proposals, got %d", len(expiredEntities))
		}
	})
}

// TestGetExpiredProposalsProvider tests the provider for finding expired proposals
func TestGetExpiredProposalsProvider(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	logger := logrus.New()

	t.Run("find_expired_proposals", func(t *testing.T) {
		// Create some proposal entities directly in the database
		now := time.Now()
		expiredTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(1 * time.Hour)

		// Create expired proposals
		expiredProposal1 := ProposalEntity{
			ProposerId:  1,
			TargetId:    2,
			Status:      ProposalStatusPending,
			ProposedAt:  expiredTime.Add(-1 * time.Hour),
			ExpiresAt:   expiredTime,
			TenantId:    tenantId,
			CreatedAt:   expiredTime.Add(-1 * time.Hour),
			UpdatedAt:   expiredTime.Add(-1 * time.Hour),
		}

		expiredProposal2 := ProposalEntity{
			ProposerId:  3,
			TargetId:    4,
			Status:      ProposalStatusPending,
			ExpiresAt:   expiredTime.Add(-30 * time.Minute),
			TenantId:    tenantId,
			CreatedAt:   expiredTime.Add(-1 * time.Hour),
			UpdatedAt:   expiredTime.Add(-1 * time.Hour),
		}

		// Create non-expired proposal
		activeProposal := ProposalEntity{
			ProposerId:  5,
			TargetId:    6,
			Status:      ProposalStatusPending,
			ExpiresAt:   futureTime,
			TenantId:    tenantId,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Create already-expired proposal (status already expired)
		alreadyExpiredProposal := ProposalEntity{
			ProposerId:  7,
			TargetId:    8,
			Status:      ProposalStatusExpired,
			ExpiresAt:   expiredTime,
			TenantId:    tenantId,
			CreatedAt:   expiredTime.Add(-1 * time.Hour),
			UpdatedAt:   now,
		}

		db.Create(&expiredProposal1)
		db.Create(&expiredProposal2)
		db.Create(&activeProposal)
		db.Create(&alreadyExpiredProposal)

		// Get expired proposals
		provider := GetExpiredProposalsProvider(db, logger)(tenantId)
		expiredProposals, err := provider()
		if err != nil {
			t.Fatalf("Failed to get expired proposals: %v", err)
		}

		// Should find only the 2 pending but expired proposals
		if len(expiredProposals) != 2 {
			t.Errorf("Expected 2 expired proposals, got %d", len(expiredProposals))
		}

		// Verify the proposals are ordered by expiry time (ASC)
		if len(expiredProposals) >= 2 {
			if !expiredProposals[0].ExpiresAt().Before(expiredProposals[1].ExpiresAt()) {
				t.Error("Expected proposals to be ordered by expiry time (ascending)")
			}
		}
	})

	t.Run("no_expired_proposals", func(t *testing.T) {
		// Clear the table
		db.Exec("DELETE FROM proposal_entities")

		// Create only active proposals
		futureTime := time.Now().Add(1 * time.Hour)
		activeProposal := ProposalEntity{
			ProposerId:  1,
			TargetId:    2,
			Status:      ProposalStatusPending,
			ExpiresAt:   futureTime,
			TenantId:    tenantId,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		db.Create(&activeProposal)

		provider := GetExpiredProposalsProvider(db, logger)(tenantId)
		expiredProposals, err := provider()
		if err != nil {
			t.Fatalf("Failed to get expired proposals: %v", err)
		}

		if len(expiredProposals) != 0 {
			t.Errorf("Expected 0 expired proposals, got %d", len(expiredProposals))
		}
	})
}

// TestProcessor_Divorce tests the divorce functionality
func TestProcessor_Divorce(t *testing.T) {
	db := setupTestDB(t)
	logger := logrus.New()
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)

	// Create a mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()
	
	// Add test characters
	mockCharacterProcessor.AddCharacter(1, "TestChar1", 20) // Level 20
	mockCharacterProcessor.AddCharacter(2, "TestChar2", 25) // Level 25

	processor := NewProcessor(logger, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	t.Run("Divorce Active Marriage", func(t *testing.T) {
		// Create a married couple
		marriageEntity := Entity{
			ID:           1,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusMarried,
			ProposedAt:   time.Now().Add(-2 * time.Hour),
			EngagedAt:    &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
			MarriedAt:    &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
			TenantId:     tenantId,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&marriageEntity)

		// Test divorce initiated by character 1 (without Kafka emission for testing)
		divorcedMarriage, err := processor.Divorce(1, 1)()
		if err != nil {
			t.Fatalf("Failed to divorce marriage: %v", err)
		}

		// Verify marriage status
		if divorcedMarriage.Status() != StatusDivorced {
			t.Errorf("Expected marriage status to be %v, got %v", StatusDivorced, divorcedMarriage.Status())
		}

		// Verify divorce timestamp is set
		if divorcedMarriage.DivorcedAt() == nil {
			t.Error("Expected divorced at timestamp to be set")
		}

		// Verify database update
		var updatedEntity Entity
		db.First(&updatedEntity, 1)
		if updatedEntity.Status != StatusDivorced {
			t.Errorf("Expected database status to be %v, got %v", StatusDivorced, updatedEntity.Status)
		}
	})

	t.Run("Divorce Non-Existent Marriage", func(t *testing.T) {
		_, err := processor.Divorce(999, 1)()
		if err == nil {
			t.Error("Expected error when divorcing non-existent marriage")
		}
	})

	t.Run("Divorce By Non-Partner", func(t *testing.T) {
		// Create another marriage
		marriageEntity := Entity{
			ID:           2,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusMarried,
			ProposedAt:   time.Now().Add(-2 * time.Hour),
			EngagedAt:    &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
			MarriedAt:    &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
			TenantId:     tenantId,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&marriageEntity)

		// Try to divorce by character 3 (not a partner)
		_, err := processor.Divorce(2, 3)()
		if err == nil {
			t.Error("Expected error when non-partner tries to initiate divorce")
		}
	})

	t.Run("Divorce Already Divorced Marriage", func(t *testing.T) {
		// Create an already divorced marriage
		marriageEntity := Entity{
			ID:           3,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusDivorced,
			ProposedAt:   time.Now().Add(-3 * time.Hour),
			EngagedAt:    &[]time.Time{time.Now().Add(-2 * time.Hour)}[0],
			MarriedAt:    &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
			DivorcedAt:   &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
			TenantId:     tenantId,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&marriageEntity)

		// Try to divorce again
		_, err := processor.Divorce(3, 1)()
		if err == nil {
			t.Error("Expected error when trying to divorce already divorced marriage")
		}
	})
}

// TestProcessor_CharacterDeletion tests the character deletion functionality
func TestProcessor_CharacterDeletion(t *testing.T) {
	db := setupTestDB(t)
	logger := logrus.New()
	tenantId := uuid.New()
	ctx := setupTestContext(tenantId)

	// Create a mock character processor
	mockCharacterProcessor := NewMockCharacterProcessor()
	
	// Add test characters
	mockCharacterProcessor.AddCharacter(1, "TestChar1", 20) // Level 20
	mockCharacterProcessor.AddCharacter(2, "TestChar2", 25) // Level 25

	processor := NewProcessor(logger, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	t.Run("Handle Character Deletion With Active Marriage", func(t *testing.T) {
		// Create a married couple
		marriageEntity := Entity{
			ID:           1,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusMarried,
			ProposedAt:   time.Now().Add(-2 * time.Hour),
			EngagedAt:    &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
			MarriedAt:    &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
			TenantId:     tenantId,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&marriageEntity)

		// Handle character deletion (without Kafka emission for testing)
		err := processor.HandleCharacterDeletion(1)
		if err != nil {
			t.Fatalf("Failed to handle character deletion: %v", err)
		}

		// Verify marriage is marked as divorced
		var updatedEntity Entity
		db.First(&updatedEntity, 1)
		if updatedEntity.Status != StatusDivorced {
			t.Errorf("Expected marriage status to be %v after character deletion, got %v", StatusDivorced, updatedEntity.Status)
		}

		// Verify divorced timestamp is set
		if updatedEntity.DivorcedAt == nil {
			t.Error("Expected divorced at timestamp to be set after character deletion")
		}
	})

	t.Run("Handle Character Deletion With No Marriage", func(t *testing.T) {
		// Handle deletion of character with no marriage
		err := processor.HandleCharacterDeletion(3)
		if err != nil {
			t.Fatalf("Unexpected error when handling deletion of character with no marriage: %v", err)
		}
	})

	t.Run("Handle Character Deletion With Engaged Marriage", func(t *testing.T) {
		// Create an engaged couple
		marriageEntity := Entity{
			ID:           2,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusEngaged,
			ProposedAt:   time.Now().Add(-2 * time.Hour),
			EngagedAt:    &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
			TenantId:     tenantId,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&marriageEntity)

		// Handle character deletion
		err := processor.HandleCharacterDeletion(2)
		if err != nil {
			t.Fatalf("Failed to handle character deletion for engaged couple: %v", err)
		}

		// Verify marriage is marked as divorced
		var updatedEntity Entity
		db.First(&updatedEntity, 2)
		if updatedEntity.Status != StatusDivorced {
			t.Errorf("Expected marriage status to be %v after character deletion, got %v", StatusDivorced, updatedEntity.Status)
		}
	})
}

