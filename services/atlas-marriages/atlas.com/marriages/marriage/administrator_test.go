package marriage

import (
	"atlas-marriages/test"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProposal(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	t.Run("successful proposal creation", func(t *testing.T) {
		proposerId := uint32(1001)
		targetId := uint32(1002)
		tenantId := uuid.New()

		provider := CreateProposal(db, log)(proposerId, targetId, tenantId)
		result, err := provider()

		require.NoError(t, err)
		assert.Equal(t, proposerId, result.ProposerId)
		assert.Equal(t, targetId, result.TargetId)
		assert.Equal(t, ProposalStatusPending, result.Status)
		assert.Equal(t, tenantId, result.TenantId)
		assert.Equal(t, uint32(0), result.RejectionCount)
		assert.False(t, result.ProposedAt.IsZero())
		assert.False(t, result.ExpiresAt.IsZero())
		assert.True(t, result.ExpiresAt.After(result.ProposedAt))
	})

	t.Run("multiple proposals can be created", func(t *testing.T) {
		tenantId := uuid.New()

		// Create first proposal
		provider1 := CreateProposal(db, log)(uint32(2001), uint32(2002), tenantId)
		result1, err := provider1()
		require.NoError(t, err)

		// Create second proposal
		provider2 := CreateProposal(db, log)(uint32(3001), uint32(3002), tenantId)
		result2, err := provider2()
		require.NoError(t, err)

		// Verify they have different IDs
		assert.NotEqual(t, result1.ID, result2.ID)
	})
}

func TestUpdateProposal(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	tenantId := uuid.New()

	t.Run("successful proposal update", func(t *testing.T) {
		// First create a proposal
		createProvider := CreateProposal(db, log)(uint32(1001), uint32(1002), tenantId)
		created, err := createProvider()
		require.NoError(t, err)

		// Build an updated proposal
		respondedAt := time.Now()
		updatedProposal, err := NewProposalBuilder(created.ProposerId, created.TargetId, created.TenantId).
			SetId(created.ID).
			SetStatus(ProposalStatusAccepted).
			SetProposedAt(created.ProposedAt).
			SetRespondedAt(&respondedAt).
			SetExpiresAt(created.ExpiresAt).
			Build()
		require.NoError(t, err)

		// Update the proposal
		updateProvider := UpdateProposal(db, log)(updatedProposal)
		result, err := updateProvider()

		require.NoError(t, err)
		assert.Equal(t, created.ID, result.ID)
		assert.Equal(t, created.ProposerId, result.ProposerId)
		assert.Equal(t, created.TargetId, result.TargetId)
		assert.Equal(t, ProposalStatusAccepted, result.Status)
	})

	t.Run("update non-existent proposal succeeds without error", func(t *testing.T) {
		// Use Pending status to avoid validation requirements for RespondedAt
		nonExistentProposal, err := NewProposalBuilder(uint32(9999), uint32(9998), tenantId).
			SetId(uint32(99999)).
			SetStatus(ProposalStatusPending).
			Build()
		require.NoError(t, err)

		updateProvider := UpdateProposal(db, log)(nonExistentProposal)
		_, err = updateProvider()
		// Update of non-existent record should still succeed (GORM doesn't error on 0 rows affected)
		assert.NoError(t, err)
	})
}

func TestCreateMarriage(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	t.Run("successful marriage creation", func(t *testing.T) {
		characterId1 := uint32(1001)
		characterId2 := uint32(1002)
		tenantId := uuid.New()

		provider := CreateMarriage(db, log)(characterId1, characterId2, tenantId)
		result, err := provider()

		require.NoError(t, err)
		assert.Equal(t, characterId1, result.CharacterId1)
		assert.Equal(t, characterId2, result.CharacterId2)
		assert.Equal(t, StatusProposed, result.Status)
		assert.Equal(t, tenantId, result.TenantId)
		assert.False(t, result.ProposedAt.IsZero())
	})

	t.Run("multiple marriages can be created", func(t *testing.T) {
		tenantId := uuid.New()

		// Create first marriage
		provider1 := CreateMarriage(db, log)(uint32(2001), uint32(2002), tenantId)
		result1, err := provider1()
		require.NoError(t, err)

		// Create second marriage
		provider2 := CreateMarriage(db, log)(uint32(3001), uint32(3002), tenantId)
		result2, err := provider2()
		require.NoError(t, err)

		// Verify they have different IDs
		assert.NotEqual(t, result1.ID, result2.ID)
	})
}

func TestUpdateMarriage(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	tenantId := uuid.New()

	t.Run("successful marriage update", func(t *testing.T) {
		// First create a marriage
		createProvider := CreateMarriage(db, log)(uint32(1001), uint32(1002), tenantId)
		created, err := createProvider()
		require.NoError(t, err)

		// Build an updated marriage
		engagedAt := time.Now()
		marriedAt := time.Now().Add(time.Hour)
		updatedMarriage, err := NewBuilder(created.CharacterId1, created.CharacterId2, created.TenantId).
			SetId(created.ID).
			SetStatus(StatusMarried).
			SetProposedAt(created.ProposedAt).
			SetEngagedAt(&engagedAt).
			SetMarriedAt(&marriedAt).
			Build()
		require.NoError(t, err)

		// Update the marriage
		updateProvider := UpdateMarriage(db, log)(updatedMarriage)
		result, err := updateProvider()

		require.NoError(t, err)
		assert.Equal(t, created.ID, result.ID)
		assert.Equal(t, created.CharacterId1, result.CharacterId1)
		assert.Equal(t, created.CharacterId2, result.CharacterId2)
		assert.Equal(t, StatusMarried, result.Status)
	})
}

func TestCreateCeremony(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	tenantId := uuid.New()
	scheduledAt := time.Now().Add(2 * time.Hour)
	invitees := []uint32{uint32(2001), uint32(2002), uint32(2003)}

	t.Run("successful ceremony creation", func(t *testing.T) {
		marriageId := uint32(789)
		characterId1 := uint32(1001)
		characterId2 := uint32(1002)

		provider := CreateCeremony(db, log)(marriageId, characterId1, characterId2, scheduledAt, invitees, tenantId)
		result, err := provider()

		require.NoError(t, err)
		assert.Equal(t, marriageId, result.MarriageId)
		assert.Equal(t, characterId1, result.CharacterId1)
		assert.Equal(t, characterId2, result.CharacterId2)
		assert.Equal(t, CeremonyStatusScheduled, result.Status)
		assert.Equal(t, tenantId, result.TenantId)
		assert.False(t, result.ScheduledAt.IsZero())
	})

	t.Run("ceremony with empty invitees", func(t *testing.T) {
		provider := CreateCeremony(db, log)(uint32(790), uint32(1003), uint32(1004), scheduledAt, []uint32{}, tenantId)
		result, err := provider()

		require.NoError(t, err)
		assert.Equal(t, CeremonyStatusScheduled, result.Status)
	})
}

func TestUpdateCeremony(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	tenantId := uuid.New()
	scheduledAt := time.Now().Add(2 * time.Hour)

	t.Run("successful ceremony update", func(t *testing.T) {
		// First create a ceremony
		createProvider := CreateCeremony(db, log)(uint32(789), uint32(1001), uint32(1002), scheduledAt, []uint32{}, tenantId)
		created, err := createProvider()
		require.NoError(t, err)

		// Build an updated ceremony entity
		startedAt := time.Now()
		updatedEntity := CeremonyEntity{
			ID:           created.ID,
			MarriageId:   created.MarriageId,
			CharacterId1: created.CharacterId1,
			CharacterId2: created.CharacterId2,
			Status:       CeremonyStatusActive,
			ScheduledAt:  created.ScheduledAt,
			StartedAt:    &startedAt,
			TenantId:     created.TenantId,
			CreatedAt:    created.CreatedAt,
			UpdatedAt:    created.UpdatedAt,
		}

		// Update the ceremony
		updateProvider := UpdateCeremony(db, log)(created.ID, updatedEntity)
		result, err := updateProvider()

		require.NoError(t, err)
		assert.Equal(t, created.ID, result.ID)
		assert.Equal(t, created.MarriageId, result.MarriageId)
		assert.Equal(t, CeremonyStatusActive, result.Status)
	})
}

func TestAdministratorFunctionPatterns(t *testing.T) {
	db := test.SetupTestDB(t, Migration)
	defer test.CleanupTestDB(t, db)

	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	t.Run("curried function pattern", func(t *testing.T) {
		// Test CreateProposal currying
		createProposalFunc := CreateProposal(db, log)
		assert.NotNil(t, createProposalFunc)

		proposalProvider := createProposalFunc(uint32(1), uint32(2), uuid.New())
		assert.NotNil(t, proposalProvider)

		// Test UpdateProposal currying
		updateProposalFunc := UpdateProposal(db, log)
		assert.NotNil(t, updateProposalFunc)

		// Test CreateMarriage currying
		createMarriageFunc := CreateMarriage(db, log)
		assert.NotNil(t, createMarriageFunc)

		marriageProvider := createMarriageFunc(uint32(1), uint32(2), uuid.New())
		assert.NotNil(t, marriageProvider)

		// Test UpdateMarriage currying
		updateMarriageFunc := UpdateMarriage(db, log)
		assert.NotNil(t, updateMarriageFunc)

		// Test CreateCeremony currying
		createCeremonyFunc := CreateCeremony(db, log)
		assert.NotNil(t, createCeremonyFunc)

		ceremonyProvider := createCeremonyFunc(uint32(1), uint32(2), uint32(3), time.Now(), []uint32{}, uuid.New())
		assert.NotNil(t, ceremonyProvider)

		// Test UpdateCeremony currying
		updateCeremonyFunc := UpdateCeremony(db, log)
		assert.NotNil(t, updateCeremonyFunc)
	})

	t.Run("provider pattern compliance", func(t *testing.T) {
		tenantId := uuid.New()

		// All providers should execute successfully with SQLite
		proposalProvider := CreateProposal(db, log)(uint32(100), uint32(200), tenantId)
		proposal, err := proposalProvider()
		require.NoError(t, err)
		assert.NotZero(t, proposal.ID)

		marriageProvider := CreateMarriage(db, log)(uint32(100), uint32(200), tenantId)
		marriage, err := marriageProvider()
		require.NoError(t, err)
		assert.NotZero(t, marriage.ID)

		ceremonyProvider := CreateCeremony(db, log)(uint32(1), uint32(100), uint32(200), time.Now(), []uint32{}, tenantId)
		ceremony, err := ceremonyProvider()
		require.NoError(t, err)
		assert.NotZero(t, ceremony.ID)
	})
}
