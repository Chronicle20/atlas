package marriage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMarriageStateTransitions tests all marriage state transition scenarios
func TestMarriageStateTransitions(t *testing.T) {
	tests := []struct {
		name           string
		currentState   MarriageState
		targetState    MarriageState
		canTransition  bool
		description    string
	}{
		// Proposed state transitions
		{
			name:          "proposed to engaged",
			currentState:  StateProposed,
			targetState:   StateEngaged,
			canTransition: true,
			description:   "Proposal acceptance should be allowed",
		},
		{
			name:          "proposed to expired",
			currentState:  StateProposed,
			targetState:   StateExpired,
			canTransition: true,
			description:   "Proposal expiry should be allowed",
		},
		{
			name:          "proposed to married - invalid",
			currentState:  StateProposed,
			targetState:   StateMarried,
			canTransition: false,
			description:   "Cannot skip engagement phase",
		},
		{
			name:          "proposed to divorced - invalid",
			currentState:  StateProposed,
			targetState:   StateDivorced,
			canTransition: false,
			description:   "Cannot divorce from proposal state",
		},

		// Engaged state transitions
		{
			name:          "engaged to married",
			currentState:  StateEngaged,
			targetState:   StateMarried,
			canTransition: true,
			description:   "Ceremony completion should be allowed",
		},
		{
			name:          "engaged to divorced - invalid",
			currentState:  StateEngaged,
			targetState:   StateDivorced,
			canTransition: false,
			description:   "Cannot divorce before marriage ceremony",
		},
		{
			name:          "engaged to expired - invalid",
			currentState:  StateEngaged,
			targetState:   StateExpired,
			canTransition: false,
			description:   "Engagement cannot expire",
		},

		// Married state transitions
		{
			name:          "married to divorced",
			currentState:  StateMarried,
			targetState:   StateDivorced,
			canTransition: true,
			description:   "Divorce from marriage should be allowed",
		},
		{
			name:          "married to engaged - invalid",
			currentState:  StateMarried,
			targetState:   StateEngaged,
			canTransition: false,
			description:   "Cannot revert to engagement",
		},
		{
			name:          "married to proposed - invalid",
			currentState:  StateMarried,
			targetState:   StateProposed,
			canTransition: false,
			description:   "Cannot revert to proposal",
		},

		// Terminal state transitions
		{
			name:          "divorced to married - invalid",
			currentState:  StateDivorced,
			targetState:   StateMarried,
			canTransition: false,
			description:   "Divorced is terminal state",
		},
		{
			name:          "expired to engaged - invalid",
			currentState:  StateExpired,
			targetState:   StateEngaged,
			canTransition: false,
			description:   "Expired is terminal state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canTransition := tt.currentState.CanTransitionTo(tt.targetState)
			if canTransition != tt.canTransition {
				t.Errorf("Expected CanTransitionTo(%v -> %v) = %v, got %v for %s", 
					tt.currentState, tt.targetState, tt.canTransition, canTransition, tt.description)
			}

			// Also test ValidTransitions
			validTransitions := tt.currentState.ValidTransitions()
			if tt.canTransition {
				found := false
				for _, valid := range validTransitions {
					if valid == tt.targetState {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected %v to be in ValidTransitions() for state %v", tt.targetState, tt.currentState)
				}
			}
		})
	}
}

// TestCeremonyStateTransitionValidation tests all ceremony state transition scenarios
func TestCeremonyStateTransitionValidation(t *testing.T) {
	tests := []struct {
		name           string
		currentState   CeremonyState
		targetState    CeremonyState
		canTransition  bool
		description    string
	}{
		// Scheduled state transitions
		{
			name:          "scheduled to active",
			currentState:  CeremonyStateScheduled,
			targetState:   CeremonyStateActive,
			canTransition: true,
			description:   "Starting ceremony should be allowed",
		},
		{
			name:          "scheduled to cancelled",
			currentState:  CeremonyStateScheduled,
			targetState:   CeremonyStateCancelled,
			canTransition: true,
			description:   "Cancelling scheduled ceremony should be allowed",
		},
		{
			name:          "scheduled to completed - invalid",
			currentState:  CeremonyStateScheduled,
			targetState:   CeremonyStateCompleted,
			canTransition: false,
			description:   "Cannot complete without starting",
		},

		// Active state transitions
		{
			name:          "active to completed",
			currentState:  CeremonyStateActive,
			targetState:   CeremonyStateCompleted,
			canTransition: true,
			description:   "Completing active ceremony should be allowed",
		},
		{
			name:          "active to cancelled",
			currentState:  CeremonyStateActive,
			targetState:   CeremonyStateCancelled,
			canTransition: true,
			description:   "Cancelling active ceremony should be allowed",
		},
		{
			name:          "active to postponed",
			currentState:  CeremonyStateActive,
			targetState:   CeremonyStatePostponed,
			canTransition: true,
			description:   "Postponing due to disconnection should be allowed",
		},

		// Postponed state transitions
		{
			name:          "postponed to scheduled",
			currentState:  CeremonyStatePostponed,
			targetState:   CeremonyStateScheduled,
			canTransition: true,
			description:   "Rescheduling postponed ceremony should be allowed",
		},
		{
			name:          "postponed to active",
			currentState:  CeremonyStatePostponed,
			targetState:   CeremonyStateActive,
			canTransition: true,
			description:   "Resuming postponed ceremony should be allowed",
		},
		{
			name:          "postponed to cancelled",
			currentState:  CeremonyStatePostponed,
			targetState:   CeremonyStateCancelled,
			canTransition: true,
			description:   "Cancelling postponed ceremony should be allowed",
		},

		// Terminal state transitions
		{
			name:          "completed to active - invalid",
			currentState:  CeremonyStateCompleted,
			targetState:   CeremonyStateActive,
			canTransition: false,
			description:   "Completed is terminal state",
		},
		{
			name:          "cancelled to scheduled - invalid",
			currentState:  CeremonyStateCancelled,
			targetState:   CeremonyStateScheduled,
			canTransition: false,
			description:   "Cancelled is terminal state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canTransition := tt.currentState.CanTransitionTo(tt.targetState)
			if canTransition != tt.canTransition {
				t.Errorf("Expected CanTransitionTo(%v -> %v) = %v, got %v for %s", 
					tt.currentState, tt.targetState, tt.canTransition, canTransition, tt.description)
			}

			// Also test ValidTransitions
			validTransitions := tt.currentState.ValidTransitions()
			if tt.canTransition {
				found := false
				for _, valid := range validTransitions {
					if valid == tt.targetState {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected %v to be in ValidTransitions() for state %v", tt.targetState, tt.currentState)
				}
			}
		})
	}
}

// TestMarriageStateHelperMethods tests IsActive, IsTerminated helper methods
func TestMarriageStateHelperMethods(t *testing.T) {
	tests := []struct {
		name          string
		state         MarriageState
		isActive      bool
		isTerminated  bool
		description   string
	}{
		{
			name:         "proposed state",
			state:        StateProposed,
			isActive:     false,
			isTerminated: false,
			description:  "Proposal is neither active nor terminated",
		},
		{
			name:         "engaged state",
			state:        StateEngaged,
			isActive:     false,
			isTerminated: false,
			description:  "Engagement is neither active nor terminated",
		},
		{
			name:         "married state",
			state:        StateMarried,
			isActive:     true,
			isTerminated: false,
			description:  "Marriage is active",
		},
		{
			name:         "divorced state",
			state:        StateDivorced,
			isActive:     false,
			isTerminated: true,
			description:  "Divorce is terminated",
		},
		{
			name:         "expired state",
			state:        StateExpired,
			isActive:     false,
			isTerminated: true,
			description:  "Expiry is terminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsActive() != tt.isActive {
				t.Errorf("Expected IsActive() = %v for %v, got %v", tt.isActive, tt.state, tt.state.IsActive())
			}
			if tt.state.IsTerminated() != tt.isTerminated {
				t.Errorf("Expected IsTerminated() = %v for %v, got %v", tt.isTerminated, tt.state, tt.state.IsTerminated())
			}
		})
	}
}

// TestCeremonyStateHelperMethods tests IsActive, IsTerminated, IsInProgress helper methods
func TestCeremonyStateHelperMethods(t *testing.T) {
	tests := []struct {
		name          string
		state         CeremonyState
		isActive      bool
		isTerminated  bool
		isInProgress  bool
		description   string
	}{
		{
			name:         "scheduled state",
			state:        CeremonyStateScheduled,
			isActive:     false,
			isTerminated: false,
			isInProgress: true,
			description:  "Scheduled ceremony is in progress but not active",
		},
		{
			name:         "active state",
			state:        CeremonyStateActive,
			isActive:     true,
			isTerminated: false,
			isInProgress: true,
			description:  "Active ceremony is active and in progress",
		},
		{
			name:         "completed state",
			state:        CeremonyStateCompleted,
			isActive:     false,
			isTerminated: true,
			isInProgress: false,
			description:  "Completed ceremony is terminated",
		},
		{
			name:         "cancelled state",
			state:        CeremonyStateCancelled,
			isActive:     false,
			isTerminated: true,
			isInProgress: false,
			description:  "Cancelled ceremony is terminated",
		},
		{
			name:         "postponed state",
			state:        CeremonyStatePostponed,
			isActive:     false,
			isTerminated: false,
			isInProgress: false,
			description:  "Postponed ceremony is neither active nor in progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsActive() != tt.isActive {
				t.Errorf("Expected IsActive() = %v for %v, got %v", tt.isActive, tt.state, tt.state.IsActive())
			}
			if tt.state.IsTerminated() != tt.isTerminated {
				t.Errorf("Expected IsTerminated() = %v for %v, got %v", tt.isTerminated, tt.state, tt.state.IsTerminated())
			}
			if tt.state.IsInProgress() != tt.isInProgress {
				t.Errorf("Expected IsInProgress() = %v for %v, got %v", tt.isInProgress, tt.state, tt.state.IsInProgress())
			}
		})
	}
}

// TestInvalidStateTransitionErrors tests processor error handling for invalid state transitions
func TestInvalidStateTransitionErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	log := logrus.New()

	processor := NewProcessor(log, ctx, db)

	t.Run("cannot accept already accepted proposal", func(t *testing.T) {
		// Create an already accepted proposal
		now := time.Now()
		marriageEntity := Entity{
			ID:           1,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       StatusEngaged,
			ProposedAt:   now,
			EngagedAt:    &now,
			TenantId:     tenantId,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err = db.Create(&marriageEntity).Error
		if err != nil {
			t.Fatalf("Failed to create marriage entity: %v", err)
		}

		proposalEntity := ProposalEntity{
			ID:         1,
			ProposerId: 1,
			TargetId:   2,
			Status:     ProposalStatusAccepted,
			ProposedAt: now,
			RespondedAt: &now,
			ExpiresAt:  now.Add(24 * time.Hour),
			TenantId:   tenantId,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		err = db.Create(&proposalEntity).Error
		if err != nil {
			t.Fatalf("Failed to create proposal entity: %v", err)
		}

		// Try to accept it again - should fail
		_, err = processor.AcceptProposal(1)()
		if err == nil {
			t.Error("Expected error when trying to accept already accepted proposal")
		}
	})

	t.Run("cannot reject already accepted proposal", func(t *testing.T) {
		// Create a proposal directly and test state transition
		tenantId2 := uuid.New()
		now := time.Now()
		acceptedProposal, err := NewProposalBuilder(1, 2, tenantId2).
			SetStatus(ProposalStatusAccepted).
			SetProposedAt(now).
			SetRespondedAt(&now).
			SetExpiresAt(now.Add(24 * time.Hour)).
			Build()
		if err != nil {
			t.Fatalf("Failed to create accepted proposal: %v", err)
		}
		
		_, err = acceptedProposal.Reject()
		if err == nil {
			t.Error("Expected error when trying to reject already accepted proposal")
		}
	})

	t.Run("cannot expire non-pending proposal", func(t *testing.T) {
		// Try to expire the already accepted proposal - should fail
		_, err = processor.ExpireProposal(1)()
		if err == nil {
			t.Error("Expected error when trying to expire non-pending proposal")
		}
	})

	t.Run("cannot start completed ceremony", func(t *testing.T) {
		// Create a completed ceremony
		now := time.Now()
		ceremonyEntity := CeremonyEntity{
			ID:           1,
			MarriageId:   1,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       CeremonyStatusCompleted,
			ScheduledAt:  now,
			StartedAt:    &now,
			CompletedAt:  &now,
			TenantId:     tenantId,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err = db.Create(&ceremonyEntity).Error
		if err != nil {
			t.Fatalf("Failed to create ceremony entity: %v", err)
		}

		// Try to start it - should fail
		_, err = processor.StartCeremony(1)()
		if err == nil {
			t.Error("Expected error when trying to start completed ceremony")
		}
	})

	t.Run("cannot complete scheduled ceremony", func(t *testing.T) {
		// Create a scheduled ceremony
		now := time.Now()
		ceremonyEntity2 := CeremonyEntity{
			ID:           2,
			MarriageId:   1,
			CharacterId1: 1,
			CharacterId2: 2,
			Status:       CeremonyStatusScheduled,
			ScheduledAt:  now,
			TenantId:     tenantId,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err = db.Create(&ceremonyEntity2).Error
		if err != nil {
			t.Fatalf("Failed to create ceremony entity: %v", err)
		}

		// Try to complete it without starting - should fail
		_, err = processor.CompleteCeremony(2)()
		if err == nil {
			t.Error("Expected error when trying to complete scheduled ceremony")
		}
	})

	t.Run("cannot divorce non-existent marriage", func(t *testing.T) {
		// Try to divorce non-existent marriage - should fail
		_, err = processor.Divorce(999, 1)()
		if err == nil {
			t.Error("Expected error when trying to divorce non-existent marriage")
		}
	})

	t.Run("cannot divorce by non-partner", func(t *testing.T) {
		// Try to divorce by someone who's not a partner - should fail
		_, err = processor.Divorce(1, 3)()
		if err == nil {
			t.Error("Expected error when non-partner tries to initiate divorce")
		}
	})
}

// TestProcessorErrorHandling tests various error scenarios in processor operations
func TestProcessorErrorHandling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	log := logrus.New()

	// Create mock character processor that returns errors
	mockCharacterProcessor := NewMockCharacterProcessor()
	mockCharacterProcessor.AddCharacterError(999, errors.New("character service unavailable"))

	processor := NewProcessor(log, ctx, db).WithCharacterProcessor(mockCharacterProcessor)

	t.Run("character service error in proposal", func(t *testing.T) {
		// Try to propose with character that has service error
		_, err := processor.Propose(999, 1)()
		if err == nil {
			t.Error("Expected error when character service fails")
		}
		if !strings.Contains(err.Error(), "character service unavailable") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("nonexistent proposal operations", func(t *testing.T) {
		// Test operations on non-existent proposals
		operations := []struct {
			name string
			op   func() error
		}{
			{
				name: "accept nonexistent proposal",
				op: func() error {
					_, err := processor.AcceptProposal(999)()
					return err
				},
			},
			{
				name: "reject nonexistent proposal",
				op: func() error {
					// Create a test proposal and try to reject it via domain logic
					tenantId := uuid.New()
					now := time.Now()
					proposal, err := NewProposalBuilder(998, 999, tenantId).
						SetStatus(ProposalStatusAccepted).
						SetProposedAt(now).
						SetRespondedAt(&now).
						SetExpiresAt(now.Add(24 * time.Hour)).
						Build()
					if err != nil {
						return err
					}
					_, err = proposal.Reject()
					return err
				},
			},
			{
				name: "expire nonexistent proposal",
				op: func() error {
					_, err := processor.ExpireProposal(999)()
					return err
				},
			},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				err := op.op()
				if err == nil {
					t.Errorf("Expected error for %s", op.name)
				}
			})
		}
	})

	t.Run("nonexistent ceremony operations", func(t *testing.T) {
		// Test operations on non-existent ceremonies
		operations := []struct {
			name string
			op   func() error
		}{
			{
				name: "start nonexistent ceremony",
				op: func() error {
					_, err := processor.StartCeremony(999)()
					return err
				},
			},
			{
				name: "complete nonexistent ceremony",
				op: func() error {
					_, err := processor.CompleteCeremony(999)()
					return err
				},
			},
			{
				name: "postpone nonexistent ceremony",
				op: func() error {
					_, err := processor.PostponeCeremony(999)()
					return err
				},
			},
			{
				name: "cancel nonexistent ceremony",
				op: func() error {
					_, err := processor.CancelCeremony(999)()
					return err
				},
			},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				err := op.op()
				if err == nil {
					t.Errorf("Expected error for %s", op.name)
				}
			})
		}
	})
}

// TestStateValidationInDomainModels tests that domain models enforce state validation
func TestStateValidationInDomainModels(t *testing.T) {
	t.Run("proposal builder validates state transitions", func(t *testing.T) {
		// Test that proposal builder enforces business rules
		tenantId := uuid.New()
		now := time.Now()
		
		// Create a valid proposal
		proposal, err := NewProposalBuilder(1, 2, tenantId).
			SetStatus(ProposalStatusPending).
			SetProposedAt(now).
			SetExpiresAt(now.Add(24 * time.Hour)).
			Build()
		
		if err != nil {
			t.Fatalf("Failed to build valid proposal: %v", err)
		}

		// Verify state methods work correctly
		if !proposal.IsPending() {
			t.Error("Expected proposal to be pending")
		}
		if proposal.IsAccepted() || proposal.IsRejected() || proposal.IsExpired() {
			t.Error("Expected only pending state to be true")
		}
		if !proposal.CanRespond() {
			t.Error("Expected pending proposal to allow responses")
		}
	})

	t.Run("marriage builder validates state transitions", func(t *testing.T) {
		// Test that marriage builder enforces business rules
		tenantId := uuid.New()
		now := time.Now()
		
		// Create a valid marriage
		marriage, err := NewBuilder(1, 2, tenantId).
			SetStatus(StatusEngaged).
			SetProposedAt(now).
			SetEngagedAt(&now).
			Build()
		
		if err != nil {
			t.Fatalf("Failed to build valid marriage: %v", err)
		}

		// Verify state methods work correctly
		if marriage.Status() != StatusEngaged {
			t.Error("Expected marriage to be engaged")
		}
		if marriage.IsActive() || marriage.IsDivorced() {
			t.Error("Expected only engaged state to be true")
		}
	})

	t.Run("ceremony builder validates state transitions", func(t *testing.T) {
		// Test that ceremony builder enforces business rules
		tenantId := uuid.New()
		now := time.Now()
		
		// Create a valid ceremony
		ceremony, err := NewCeremonyBuilder(1, 1, 2, tenantId).
			SetStatus(CeremonyStatusScheduled).
			SetScheduledAt(now.Add(time.Hour)).
			SetInvitees([]uint32{3, 4, 5}).
			Build()
		
		if err != nil {
			t.Fatalf("Failed to build valid ceremony: %v", err)
		}

		// Verify state methods work correctly
		if !ceremony.IsScheduled() {
			t.Error("Expected ceremony to be scheduled")
		}
		if ceremony.IsActive() || ceremony.IsCompleted() || ceremony.IsCancelled() {
			t.Error("Expected only scheduled state to be true")
		}
		if !ceremony.CanStart() {
			t.Error("Expected scheduled ceremony to be startable")
		}
	})
}

