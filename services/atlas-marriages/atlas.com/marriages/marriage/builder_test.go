package marriage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarriageBuilder(t *testing.T) {
	testTenantId := uuid.New()
	characterId1 := uint32(1001)
	characterId2 := uint32(1002)

	t.Run("NewBuilder creates valid builder with required parameters", func(t *testing.T) {
		builder := NewBuilder(characterId1, characterId2, testTenantId)

		marriage, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, characterId1, marriage.CharacterId1())
		assert.Equal(t, characterId2, marriage.CharacterId2())
		assert.Equal(t, testTenantId, marriage.TenantId())
		assert.Equal(t, StatusProposed, marriage.Status())
		assert.False(t, marriage.ProposedAt().IsZero())
		assert.Nil(t, marriage.EngagedAt())
		assert.Nil(t, marriage.MarriedAt())
		assert.Nil(t, marriage.DivorcedAt())
	})

	t.Run("Builder validates required fields", func(t *testing.T) {
		tests := []struct {
			name           string
			characterId1   uint32
			characterId2   uint32
			tenantId       uuid.UUID
			expectedError  string
		}{
			{
				name:          "zero character ID 1",
				characterId1:  0,
				characterId2:  characterId2,
				tenantId:      testTenantId,
				expectedError: "character ID 1 is required",
			},
			{
				name:          "zero character ID 2",
				characterId1:  characterId1,
				characterId2:  0,
				tenantId:      testTenantId,
				expectedError: "character ID 2 is required",
			},
			{
				name:          "same character IDs",
				characterId1:  characterId1,
				characterId2:  characterId1,
				tenantId:      testTenantId,
				expectedError: "character cannot marry themselves",
			},
			{
				name:          "nil tenant ID",
				characterId1:  characterId1,
				characterId2:  characterId2,
				tenantId:      uuid.Nil,
				expectedError: "tenant ID is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewBuilder(tt.characterId1, tt.characterId2, tt.tenantId)
				
				_, err := builder.Build()
				
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Builder fluent setters work correctly", func(t *testing.T) {
		id := uint32(12345)
		proposedAt := time.Now().Add(-time.Hour)
		engagedAt := time.Now().Add(-30 * time.Minute)
		marriedAt := time.Now().Add(-5 * time.Minute)
		createdAt := time.Now().Add(-2 * time.Hour)
		updatedAt := time.Now()

		builder := NewBuilder(characterId1, characterId2, testTenantId).
			SetId(id).
			SetStatus(StatusMarried).
			SetProposedAt(proposedAt).
			SetEngagedAt(&engagedAt).
			SetMarriedAt(&marriedAt).
			SetCreatedAt(createdAt).
			SetUpdatedAt(updatedAt)

		marriage, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, id, marriage.Id())
		assert.Equal(t, StatusMarried, marriage.Status())
		assert.Equal(t, proposedAt, marriage.ProposedAt())
		assert.Equal(t, engagedAt, *marriage.EngagedAt())
		assert.Equal(t, marriedAt, *marriage.MarriedAt())
		assert.Equal(t, createdAt, marriage.CreatedAt())
		assert.Equal(t, updatedAt, marriage.UpdatedAt())
	})

	t.Run("Marriage state transitions validation", func(t *testing.T) {
		tests := []struct {
			name          string
			status        MarriageStatus
			engagedAt     *time.Time
			marriedAt     *time.Time
			divorcedAt    *time.Time
			expectedError string
		}{
			{
				name:          "proposed with engagement timestamp",
				status:        StatusProposed,
				engagedAt:     timePtr(time.Now()),
				expectedError: "proposed marriage cannot have engagement timestamp",
			},
			{
				name:          "proposed with marriage timestamp",
				status:        StatusProposed,
				marriedAt:     timePtr(time.Now()),
				expectedError: "proposed marriage cannot have marriage timestamp",
			},
			{
				name:          "proposed with divorce timestamp",
				status:        StatusProposed,
				divorcedAt:    timePtr(time.Now()),
				expectedError: "proposed marriage cannot have divorce timestamp",
			},
			{
				name:          "engaged without engagement timestamp",
				status:        StatusEngaged,
				expectedError: "engaged marriage must have engagement timestamp",
			},
			{
				name:          "engaged with marriage timestamp",
				status:        StatusEngaged,
				engagedAt:     timePtr(time.Now()),
				marriedAt:     timePtr(time.Now()),
				expectedError: "engaged marriage cannot have marriage timestamp",
			},
			{
				name:          "engaged with divorce timestamp",
				status:        StatusEngaged,
				engagedAt:     timePtr(time.Now()),
				divorcedAt:    timePtr(time.Now()),
				expectedError: "engaged marriage cannot have divorce timestamp",
			},
			{
				name:          "married without engagement timestamp",
				status:        StatusMarried,
				marriedAt:     timePtr(time.Now()),
				expectedError: "married marriage must have engagement timestamp",
			},
			{
				name:          "married without marriage timestamp",
				status:        StatusMarried,
				engagedAt:     timePtr(time.Now()),
				expectedError: "married marriage must have marriage timestamp",
			},
			{
				name:          "married with divorce timestamp",
				status:        StatusMarried,
				engagedAt:     timePtr(time.Now()),
				marriedAt:     timePtr(time.Now()),
				divorcedAt:    timePtr(time.Now()),
				expectedError: "married marriage cannot have divorce timestamp",
			},
			{
				name:          "divorced without engagement timestamp",
				status:        StatusDivorced,
				marriedAt:     timePtr(time.Now()),
				divorcedAt:    timePtr(time.Now()),
				expectedError: "divorced marriage must have engagement timestamp",
			},
			{
				name:          "divorced without marriage timestamp",
				status:        StatusDivorced,
				engagedAt:     timePtr(time.Now()),
				divorcedAt:    timePtr(time.Now()),
				expectedError: "divorced marriage must have marriage timestamp",
			},
			{
				name:          "divorced without divorce timestamp",
				status:        StatusDivorced,
				engagedAt:     timePtr(time.Now()),
				marriedAt:     timePtr(time.Now()),
				expectedError: "divorced marriage must have divorce timestamp",
			},
			{
				name:          "expired with engagement timestamp",
				status:        StatusExpired,
				engagedAt:     timePtr(time.Now()),
				expectedError: "expired marriage cannot have engagement timestamp",
			},
			{
				name:          "expired with marriage timestamp",
				status:        StatusExpired,
				marriedAt:     timePtr(time.Now()),
				expectedError: "expired marriage cannot have marriage timestamp",
			},
			{
				name:          "expired with divorce timestamp",
				status:        StatusExpired,
				divorcedAt:    timePtr(time.Now()),
				expectedError: "expired marriage cannot have divorce timestamp",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewBuilder(characterId1, characterId2, testTenantId).
					SetStatus(tt.status)

				if tt.engagedAt != nil {
					builder = builder.SetEngagedAt(tt.engagedAt)
				}
				if tt.marriedAt != nil {
					builder = builder.SetMarriedAt(tt.marriedAt)
				}
				if tt.divorcedAt != nil {
					builder = builder.SetDivorcedAt(tt.divorcedAt)
				}

				_, err := builder.Build()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Valid marriage state transitions", func(t *testing.T) {
		tests := []struct {
			name       string
			status     MarriageStatus
			engagedAt  *time.Time
			marriedAt  *time.Time
			divorcedAt *time.Time
		}{
			{
				name:   "valid proposed state",
				status: StatusProposed,
			},
			{
				name:      "valid engaged state",
				status:    StatusEngaged,
				engagedAt: timePtr(time.Now()),
			},
			{
				name:      "valid married state",
				status:    StatusMarried,
				engagedAt: timePtr(time.Now().Add(-time.Hour)),
				marriedAt: timePtr(time.Now()),
			},
			{
				name:       "valid divorced state",
				status:     StatusDivorced,
				engagedAt:  timePtr(time.Now().Add(-2*time.Hour)),
				marriedAt:  timePtr(time.Now().Add(-time.Hour)),
				divorcedAt: timePtr(time.Now()),
			},
			{
				name:   "valid expired state",
				status: StatusExpired,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewBuilder(characterId1, characterId2, testTenantId).
					SetStatus(tt.status)

				if tt.engagedAt != nil {
					builder = builder.SetEngagedAt(tt.engagedAt)
				}
				if tt.marriedAt != nil {
					builder = builder.SetMarriedAt(tt.marriedAt)
				}
				if tt.divorcedAt != nil {
					builder = builder.SetDivorcedAt(tt.divorcedAt)
				}

				marriage, err := builder.Build()

				require.NoError(t, err)
				assert.Equal(t, tt.status, marriage.Status())
			})
		}
	})
}

func TestProposalBuilder(t *testing.T) {
	testTenantId := uuid.New()
	proposerId := uint32(1001)
	targetId := uint32(1002)

	t.Run("NewProposalBuilder creates valid builder with required parameters", func(t *testing.T) {
		builder := NewProposalBuilder(proposerId, targetId, testTenantId)

		proposal, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, proposerId, proposal.ProposerId())
		assert.Equal(t, targetId, proposal.TargetId())
		assert.Equal(t, testTenantId, proposal.TenantId())
		assert.Equal(t, ProposalStatusPending, proposal.Status())
		assert.False(t, proposal.ProposedAt().IsZero())
		assert.False(t, proposal.ExpiresAt().IsZero())
		assert.Equal(t, uint32(0), proposal.RejectionCount())
		assert.Nil(t, proposal.RespondedAt())
		assert.Nil(t, proposal.CooldownUntil())
	})

	t.Run("Builder validates required fields", func(t *testing.T) {
		tests := []struct {
			name          string
			proposerId    uint32
			targetId      uint32
			tenantId      uuid.UUID
			expectedError string
		}{
			{
				name:          "zero proposer ID",
				proposerId:    0,
				targetId:      targetId,
				tenantId:      testTenantId,
				expectedError: "proposer ID is required",
			},
			{
				name:          "zero target ID",
				proposerId:    proposerId,
				targetId:      0,
				tenantId:      testTenantId,
				expectedError: "target ID is required",
			},
			{
				name:          "same proposer and target IDs",
				proposerId:    proposerId,
				targetId:      proposerId,
				tenantId:      testTenantId,
				expectedError: "character cannot propose to themselves",
			},
			{
				name:          "nil tenant ID",
				proposerId:    proposerId,
				targetId:      targetId,
				tenantId:      uuid.Nil,
				expectedError: "tenant ID is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewProposalBuilder(tt.proposerId, tt.targetId, tt.tenantId)
				
				_, err := builder.Build()
				
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Builder validates expiry time", func(t *testing.T) {
		pastTime := time.Now().Add(-time.Hour)
		
		builder := NewProposalBuilder(proposerId, targetId, testTenantId).
			SetProposedAt(time.Now()).
			SetExpiresAt(pastTime)

		_, err := builder.Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expiry time cannot be before proposal time")
	})

	t.Run("Builder fluent setters work correctly", func(t *testing.T) {
		id := uint32(12345)
		proposedAt := time.Now().Add(-time.Hour)
		respondedAt := time.Now().Add(-30 * time.Minute)
		expiresAt := time.Now().Add(time.Hour)
		cooldownUntil := time.Now().Add(24 * time.Hour)
		rejectionCount := uint32(2)
		createdAt := time.Now().Add(-2 * time.Hour)
		updatedAt := time.Now()

		builder := NewProposalBuilder(proposerId, targetId, testTenantId).
			SetId(id).
			SetStatus(ProposalStatusRejected).
			SetProposedAt(proposedAt).
			SetRespondedAt(&respondedAt).
			SetExpiresAt(expiresAt).
			SetRejectionCount(rejectionCount).
			SetCooldownUntil(&cooldownUntil).
			SetCreatedAt(createdAt).
			SetUpdatedAt(updatedAt)

		proposal, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, id, proposal.Id())
		assert.Equal(t, ProposalStatusRejected, proposal.Status())
		assert.Equal(t, proposedAt, proposal.ProposedAt())
		assert.Equal(t, respondedAt, *proposal.RespondedAt())
		assert.Equal(t, expiresAt, proposal.ExpiresAt())
		assert.Equal(t, rejectionCount, proposal.RejectionCount())
		assert.Equal(t, cooldownUntil, *proposal.CooldownUntil())
		assert.Equal(t, createdAt, proposal.CreatedAt())
		assert.Equal(t, updatedAt, proposal.UpdatedAt())
	})

	t.Run("Proposal state transitions validation", func(t *testing.T) {
		tests := []struct {
			name           string
			status         ProposalStatus
			respondedAt    *time.Time
			cooldownUntil  *time.Time
			expectedError  string
		}{
			{
				name:          "pending with response timestamp",
				status:        ProposalStatusPending,
				respondedAt:   timePtr(time.Now()),
				expectedError: "pending proposal cannot have response timestamp",
			},
			{
				name:          "accepted without response timestamp",
				status:        ProposalStatusAccepted,
				expectedError: "accepted proposal must have response timestamp",
			},
			{
				name:          "rejected without response timestamp",
				status:        ProposalStatusRejected,
				cooldownUntil: timePtr(time.Now()),
				expectedError: "rejected proposal must have response timestamp",
			},
			{
				name:          "rejected without cooldown timestamp",
				status:        ProposalStatusRejected,
				respondedAt:   timePtr(time.Now()),
				expectedError: "rejected proposal must have cooldown timestamp",
			},
			{
				name:          "expired with response timestamp",
				status:        ProposalStatusExpired,
				respondedAt:   timePtr(time.Now()),
				expectedError: "expired proposal cannot have response timestamp",
			},
			{
				name:          "cancelled with response timestamp",
				status:        ProposalStatusCancelled,
				respondedAt:   timePtr(time.Now()),
				expectedError: "cancelled proposal cannot have response timestamp",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewProposalBuilder(proposerId, targetId, testTenantId).
					SetStatus(tt.status)

				if tt.respondedAt != nil {
					builder = builder.SetRespondedAt(tt.respondedAt)
				}
				if tt.cooldownUntil != nil {
					builder = builder.SetCooldownUntil(tt.cooldownUntil)
				}

				_, err := builder.Build()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Valid proposal state transitions", func(t *testing.T) {
		tests := []struct {
			name          string
			status        ProposalStatus
			respondedAt   *time.Time
			cooldownUntil *time.Time
		}{
			{
				name:   "valid pending state",
				status: ProposalStatusPending,
			},
			{
				name:        "valid accepted state",
				status:      ProposalStatusAccepted,
				respondedAt: timePtr(time.Now()),
			},
			{
				name:          "valid rejected state",
				status:        ProposalStatusRejected,
				respondedAt:   timePtr(time.Now()),
				cooldownUntil: timePtr(time.Now().Add(24 * time.Hour)),
			},
			{
				name:   "valid expired state",
				status: ProposalStatusExpired,
			},
			{
				name:   "valid cancelled state",
				status: ProposalStatusCancelled,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewProposalBuilder(proposerId, targetId, testTenantId).
					SetStatus(tt.status)

				if tt.respondedAt != nil {
					builder = builder.SetRespondedAt(tt.respondedAt)
				}
				if tt.cooldownUntil != nil {
					builder = builder.SetCooldownUntil(tt.cooldownUntil)
				}

				proposal, err := builder.Build()

				require.NoError(t, err)
				assert.Equal(t, tt.status, proposal.Status())
			})
		}
	})
}

func TestCeremonyBuilder(t *testing.T) {
	testTenantId := uuid.New()
	marriageId := uint32(5001)
	characterId1 := uint32(1001)
	characterId2 := uint32(1002)

	t.Run("NewCeremonyBuilder creates valid builder with required parameters", func(t *testing.T) {
		builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId)

		ceremony, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, marriageId, ceremony.MarriageId())
		assert.Equal(t, characterId1, ceremony.CharacterId1())
		assert.Equal(t, characterId2, ceremony.CharacterId2())
		assert.Equal(t, testTenantId, ceremony.TenantId())
		assert.Equal(t, CeremonyStatusScheduled, ceremony.Status())
		assert.False(t, ceremony.ScheduledAt().IsZero())
		assert.Empty(t, ceremony.Invitees())
		assert.Nil(t, ceremony.StartedAt())
		assert.Nil(t, ceremony.CompletedAt())
		assert.Nil(t, ceremony.CancelledAt())
		assert.Nil(t, ceremony.PostponedAt())
	})

	t.Run("Builder validates required fields", func(t *testing.T) {
		tests := []struct {
			name          string
			marriageId    uint32
			characterId1  uint32
			characterId2  uint32
			tenantId      uuid.UUID
			expectedError string
		}{
			{
				name:          "zero marriage ID",
				marriageId:    0,
				characterId1:  characterId1,
				characterId2:  characterId2,
				tenantId:      testTenantId,
				expectedError: "marriage ID is required",
			},
			{
				name:          "zero character ID 1",
				marriageId:    marriageId,
				characterId1:  0,
				characterId2:  characterId2,
				tenantId:      testTenantId,
				expectedError: "character ID 1 is required",
			},
			{
				name:          "zero character ID 2",
				marriageId:    marriageId,
				characterId1:  characterId1,
				characterId2:  0,
				tenantId:      testTenantId,
				expectedError: "character ID 2 is required",
			},
			{
				name:          "same character IDs",
				marriageId:    marriageId,
				characterId1:  characterId1,
				characterId2:  characterId1,
				tenantId:      testTenantId,
				expectedError: "character cannot have ceremony with themselves",
			},
			{
				name:          "nil tenant ID",
				marriageId:    marriageId,
				characterId1:  characterId1,
				characterId2:  characterId2,
				tenantId:      uuid.Nil,
				expectedError: "tenant ID is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewCeremonyBuilder(tt.marriageId, tt.characterId1, tt.characterId2, tt.tenantId)
				
				_, err := builder.Build()
				
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Builder validates invitee constraints", func(t *testing.T) {
		t.Run("too many invitees", func(t *testing.T) {
			// Create 16 invitees (max is 15)
			invitees := make([]uint32, 16)
			for i := 0; i < 16; i++ {
				invitees[i] = uint32(2000 + i)
			}

			builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
				SetInvitees(invitees)

			_, err := builder.Build()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "too many invitees")
		})

		t.Run("partners cannot be invitees", func(t *testing.T) {
			invitees := []uint32{characterId1, uint32(2001), uint32(2002)}

			builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
				SetInvitees(invitees)

			_, err := builder.Build()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "partners cannot be invitees")
		})

		t.Run("duplicate invitees", func(t *testing.T) {
			invitees := []uint32{uint32(2001), uint32(2002), uint32(2001)}

			builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
				SetInvitees(invitees)

			_, err := builder.Build()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "duplicate invitee")
		})

		t.Run("valid invitees", func(t *testing.T) {
			invitees := []uint32{uint32(2001), uint32(2002), uint32(2003)}

			builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
				SetInvitees(invitees)

			ceremony, err := builder.Build()

			require.NoError(t, err)
			assert.Equal(t, invitees, ceremony.Invitees())
		})
	})

	t.Run("Builder fluent setters work correctly", func(t *testing.T) {
		id := uint32(12345)
		scheduledAt := time.Now().Add(time.Hour)
		startedAt := time.Now().Add(2 * time.Hour)
		completedAt := time.Now().Add(3 * time.Hour)
		invitees := []uint32{uint32(2001), uint32(2002)}
		createdAt := time.Now().Add(-time.Hour)
		updatedAt := time.Now()

		builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
			SetId(id).
			SetStatus(CeremonyStatusCompleted).
			SetScheduledAt(scheduledAt).
			SetStartedAt(&startedAt).
			SetCompletedAt(&completedAt).
			SetInvitees(invitees).
			SetCreatedAt(createdAt).
			SetUpdatedAt(updatedAt)

		ceremony, err := builder.Build()

		require.NoError(t, err)
		assert.Equal(t, id, ceremony.Id())
		assert.Equal(t, CeremonyStatusCompleted, ceremony.Status())
		assert.Equal(t, scheduledAt, ceremony.ScheduledAt())
		assert.Equal(t, startedAt, *ceremony.StartedAt())
		assert.Equal(t, completedAt, *ceremony.CompletedAt())
		assert.Equal(t, invitees, ceremony.Invitees())
		assert.Equal(t, createdAt, ceremony.CreatedAt())
		assert.Equal(t, updatedAt, ceremony.UpdatedAt())
	})

	t.Run("Ceremony state transitions validation", func(t *testing.T) {
		tests := []struct {
			name          string
			status        CeremonyStatus
			startedAt     *time.Time
			completedAt   *time.Time
			cancelledAt   *time.Time
			postponedAt   *time.Time
			expectedError string
		}{
			{
				name:          "scheduled with started timestamp",
				status:        CeremonyStatusScheduled,
				startedAt:     timePtr(time.Now()),
				expectedError: "scheduled ceremony cannot have started timestamp",
			},
			{
				name:          "scheduled with completed timestamp",
				status:        CeremonyStatusScheduled,
				completedAt:   timePtr(time.Now()),
				expectedError: "scheduled ceremony cannot have completed timestamp",
			},
			{
				name:          "scheduled with cancelled timestamp",
				status:        CeremonyStatusScheduled,
				cancelledAt:   timePtr(time.Now()),
				expectedError: "scheduled ceremony cannot have cancelled timestamp",
			},
			{
				name:          "scheduled with postponed timestamp",
				status:        CeremonyStatusScheduled,
				postponedAt:   timePtr(time.Now()),
				expectedError: "scheduled ceremony cannot have postponed timestamp",
			},
			{
				name:          "active without started timestamp",
				status:        CeremonyStatusActive,
				expectedError: "active ceremony must have started timestamp",
			},
			{
				name:          "active with completed timestamp",
				status:        CeremonyStatusActive,
				startedAt:     timePtr(time.Now()),
				completedAt:   timePtr(time.Now()),
				expectedError: "active ceremony cannot have completed timestamp",
			},
			{
				name:          "active with cancelled timestamp",
				status:        CeremonyStatusActive,
				startedAt:     timePtr(time.Now()),
				cancelledAt:   timePtr(time.Now()),
				expectedError: "active ceremony cannot have cancelled timestamp",
			},
			{
				name:          "completed without started timestamp",
				status:        CeremonyStatusCompleted,
				completedAt:   timePtr(time.Now()),
				expectedError: "completed ceremony must have started timestamp",
			},
			{
				name:          "completed without completed timestamp",
				status:        CeremonyStatusCompleted,
				startedAt:     timePtr(time.Now()),
				expectedError: "completed ceremony must have completed timestamp",
			},
			{
				name:          "completed with cancelled timestamp",
				status:        CeremonyStatusCompleted,
				startedAt:     timePtr(time.Now()),
				completedAt:   timePtr(time.Now()),
				cancelledAt:   timePtr(time.Now()),
				expectedError: "completed ceremony cannot have cancelled timestamp",
			},
			{
				name:          "cancelled without cancelled timestamp",
				status:        CeremonyStatusCancelled,
				expectedError: "cancelled ceremony must have cancelled timestamp",
			},
			{
				name:          "cancelled with completed timestamp",
				status:        CeremonyStatusCancelled,
				cancelledAt:   timePtr(time.Now()),
				completedAt:   timePtr(time.Now()),
				expectedError: "cancelled ceremony cannot have completed timestamp",
			},
			{
				name:          "postponed without postponed timestamp",
				status:        CeremonyStatusPostponed,
				expectedError: "postponed ceremony must have postponed timestamp",
			},
			{
				name:          "postponed with completed timestamp",
				status:        CeremonyStatusPostponed,
				postponedAt:   timePtr(time.Now()),
				completedAt:   timePtr(time.Now()),
				expectedError: "postponed ceremony cannot have completed timestamp",
			},
			{
				name:          "postponed with cancelled timestamp",
				status:        CeremonyStatusPostponed,
				postponedAt:   timePtr(time.Now()),
				cancelledAt:   timePtr(time.Now()),
				expectedError: "postponed ceremony cannot have cancelled timestamp",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
					SetStatus(tt.status)

				if tt.startedAt != nil {
					builder = builder.SetStartedAt(tt.startedAt)
				}
				if tt.completedAt != nil {
					builder = builder.SetCompletedAt(tt.completedAt)
				}
				if tt.cancelledAt != nil {
					builder = builder.SetCancelledAt(tt.cancelledAt)
				}
				if tt.postponedAt != nil {
					builder = builder.SetPostponedAt(tt.postponedAt)
				}

				_, err := builder.Build()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Valid ceremony state transitions", func(t *testing.T) {
		tests := []struct {
			name        string
			status      CeremonyStatus
			startedAt   *time.Time
			completedAt *time.Time
			cancelledAt *time.Time
			postponedAt *time.Time
		}{
			{
				name:   "valid scheduled state",
				status: CeremonyStatusScheduled,
			},
			{
				name:      "valid active state",
				status:    CeremonyStatusActive,
				startedAt: timePtr(time.Now()),
			},
			{
				name:        "valid completed state",
				status:      CeremonyStatusCompleted,
				startedAt:   timePtr(time.Now().Add(-time.Hour)),
				completedAt: timePtr(time.Now()),
			},
			{
				name:        "valid cancelled state",
				status:      CeremonyStatusCancelled,
				cancelledAt: timePtr(time.Now()),
			},
			{
				name:        "valid postponed state",
				status:      CeremonyStatusPostponed,
				postponedAt: timePtr(time.Now()),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
					SetStatus(tt.status)

				if tt.startedAt != nil {
					builder = builder.SetStartedAt(tt.startedAt)
				}
				if tt.completedAt != nil {
					builder = builder.SetCompletedAt(tt.completedAt)
				}
				if tt.cancelledAt != nil {
					builder = builder.SetCancelledAt(tt.cancelledAt)
				}
				if tt.postponedAt != nil {
					builder = builder.SetPostponedAt(tt.postponedAt)
				}

				ceremony, err := builder.Build()

				require.NoError(t, err)
				assert.Equal(t, tt.status, ceremony.Status())
			})
		}
	})

	t.Run("Invitees slice immutability", func(t *testing.T) {
		originalInvitees := []uint32{uint32(2001), uint32(2002)}
		
		builder := NewCeremonyBuilder(marriageId, characterId1, characterId2, testTenantId).
			SetInvitees(originalInvitees)

		ceremony, err := builder.Build()
		require.NoError(t, err)

		// Modify original slice
		originalInvitees[0] = uint32(9999)

		// Ceremony should be unaffected
		assert.Equal(t, uint32(2001), ceremony.Invitees()[0])
		assert.NotEqual(t, originalInvitees[0], ceremony.Invitees()[0])
	})
}

// Helper function to create a time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}