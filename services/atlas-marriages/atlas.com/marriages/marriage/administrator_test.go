package marriage

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateProposal(t *testing.T) {
	tests := []struct {
		name              string
		proposerId        uint32
		targetId          uint32
		tenantId          uuid.UUID
		mockExpectations  func(sqlmock.Sqlmock)
		expectedError     bool
		expectedProposer  uint32
		expectedTarget    uint32
		expectedStatus    ProposalStatus
	}{
		{
			name:       "successful proposal creation",
			proposerId: uint32(1001),
			targetId:   uint32(1002),
			tenantId:   uuid.New(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "proposals"`)).
					WithArgs(
						uint32(1001),     // proposer_id
						uint32(1002),     // target_id
						ProposalStatusPending, // status
						sqlmock.AnyArg(), // proposed_at
						sqlmock.AnyArg(), // responded_at (nil)
						sqlmock.AnyArg(), // expires_at
						uint32(0),        // rejection_count
						sqlmock.AnyArg(), // cooldown_until (nil)
						sqlmock.AnyArg(), // tenant_id
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			expectedError:    false,
			expectedProposer: uint32(1001),
			expectedTarget:   uint32(1002),
			expectedStatus:   ProposalStatusPending,
		},
		{
			name:       "database error during creation",
			proposerId: uint32(1001),
			targetId:   uint32(1002),
			tenantId:   uuid.New(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "proposals"`)).
					WithArgs(
						uint32(1001),
						uint32(1002),
						ProposalStatusPending,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						uint32(0),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel) // Suppress log output during tests

			// Execute the function
			provider := CreateProposal(gormDB, log)(tt.proposerId, tt.targetId, tt.tenantId)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedProposer, result.ProposerId)
				assert.Equal(t, tt.expectedTarget, result.TargetId)
				assert.Equal(t, tt.expectedStatus, result.Status)
				assert.Equal(t, tt.tenantId, result.TenantId)
				assert.Equal(t, uint32(0), result.RejectionCount)
				assert.False(t, result.ProposedAt.IsZero())
				assert.False(t, result.ExpiresAt.IsZero())
				assert.True(t, result.ExpiresAt.After(result.ProposedAt))
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestUpdateProposal(t *testing.T) {
	tenantId := uuid.New()
	proposedAt := time.Now().Add(-time.Hour)
	respondedAt := time.Now()

	tests := []struct {
		name             string
		proposal         Proposal
		mockExpectations func(sqlmock.Sqlmock)
		expectedError    bool
	}{
		{
			name: "successful proposal update",
			proposal: func() Proposal {
				proposal, _ := NewProposalBuilder(uint32(1001), uint32(1002), tenantId).
					SetId(uint32(123)).
					SetStatus(ProposalStatusAccepted).
					SetProposedAt(proposedAt).
					SetRespondedAt(&respondedAt).
					Build()
				return proposal
			}(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "proposals" SET`)).
					WithArgs(
						uint32(1001),             // proposer_id
						uint32(1002),             // target_id
						ProposalStatusAccepted,   // status
						sqlmock.AnyArg(),         // proposed_at
						sqlmock.AnyArg(),         // responded_at
						sqlmock.AnyArg(),         // expires_at
						uint32(0),                // rejection_count
						sqlmock.AnyArg(),         // cooldown_until
						tenantId,                 // tenant_id
						sqlmock.AnyArg(),         // created_at
						sqlmock.AnyArg(),         // updated_at
						uint32(123),              // id
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name: "database error during update",
			proposal: func() Proposal {
				proposal, _ := NewProposalBuilder(uint32(1001), uint32(1002), tenantId).
					SetId(uint32(123)).
					SetStatus(ProposalStatusRejected).
					Build()
				return proposal
			}(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "proposals" SET`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						uint32(123),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel)

			// Execute the function
			provider := UpdateProposal(gormDB, log)(tt.proposal)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.proposal.Id(), result.ID)
				assert.Equal(t, tt.proposal.ProposerId(), result.ProposerId)
				assert.Equal(t, tt.proposal.TargetId(), result.TargetId)
				assert.Equal(t, tt.proposal.Status(), result.Status)
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestCreateMarriage(t *testing.T) {
	tests := []struct {
		name              string
		characterId1      uint32
		characterId2      uint32
		tenantId          uuid.UUID
		mockExpectations  func(sqlmock.Sqlmock)
		expectedError     bool
		expectedChar1     uint32
		expectedChar2     uint32
		expectedStatus    MarriageStatus
	}{
		{
			name:         "successful marriage creation",
			characterId1: uint32(1001),
			characterId2: uint32(1002),
			tenantId:     uuid.New(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "marriages"`)).
					WithArgs(
						uint32(1001),     // character_id_1
						uint32(1002),     // character_id_2
						StatusProposed,   // status
						sqlmock.AnyArg(), // proposed_at
						sqlmock.AnyArg(), // engaged_at (nil)
						sqlmock.AnyArg(), // married_at (nil)
						sqlmock.AnyArg(), // divorced_at (nil)
						sqlmock.AnyArg(), // tenant_id
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			expectedError:  false,
			expectedChar1:  uint32(1001),
			expectedChar2:  uint32(1002),
			expectedStatus: StatusProposed,
		},
		{
			name:         "database error during creation",
			characterId1: uint32(1001),
			characterId2: uint32(1002),
			tenantId:     uuid.New(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "marriages"`)).
					WithArgs(
						uint32(1001),
						uint32(1002),
						StatusProposed,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel)

			// Execute the function
			provider := CreateMarriage(gormDB, log)(tt.characterId1, tt.characterId2, tt.tenantId)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedChar1, result.CharacterId1)
				assert.Equal(t, tt.expectedChar2, result.CharacterId2)
				assert.Equal(t, tt.expectedStatus, result.Status)
				assert.Equal(t, tt.tenantId, result.TenantId)
				assert.False(t, result.ProposedAt.IsZero())
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestUpdateMarriage(t *testing.T) {
	tenantId := uuid.New()
	proposedAt := time.Now().Add(-2*time.Hour)
	engagedAt := time.Now().Add(-time.Hour)
	marriedAt := time.Now()

	tests := []struct {
		name             string
		marriage         Marriage
		mockExpectations func(sqlmock.Sqlmock)
		expectedError    bool
	}{
		{
			name: "successful marriage update",
			marriage: func() Marriage {
				marriage, _ := NewBuilder(uint32(1001), uint32(1002), tenantId).
					SetId(uint32(456)).
					SetStatus(StatusMarried).
					SetProposedAt(proposedAt).
					SetEngagedAt(&engagedAt).
					SetMarriedAt(&marriedAt).
					Build()
				return marriage
			}(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "marriages" SET`)).
					WithArgs(
						uint32(1001),   // character_id_1
						uint32(1002),   // character_id_2
						StatusMarried,  // status
						sqlmock.AnyArg(), // proposed_at
						sqlmock.AnyArg(), // engaged_at
						sqlmock.AnyArg(), // married_at
						sqlmock.AnyArg(), // divorced_at
						tenantId,         // tenant_id
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						uint32(456),      // id
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name: "database error during update",
			marriage: func() Marriage {
				marriage, _ := NewBuilder(uint32(1001), uint32(1002), tenantId).
					SetId(uint32(456)).
					SetStatus(StatusDivorced).
					Build()
				return marriage
			}(),
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "marriages" SET`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						uint32(456),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel)

			// Execute the function
			provider := UpdateMarriage(gormDB, log)(tt.marriage)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.marriage.Id(), result.ID)
				assert.Equal(t, tt.marriage.CharacterId1(), result.CharacterId1)
				assert.Equal(t, tt.marriage.CharacterId2(), result.CharacterId2)
				assert.Equal(t, tt.marriage.Status(), result.Status)
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestCreateCeremony(t *testing.T) {
	tenantId := uuid.New()
	scheduledAt := time.Now().Add(2 * time.Hour)
	invitees := []uint32{uint32(2001), uint32(2002), uint32(2003)}

	tests := []struct {
		name              string
		marriageId        uint32
		characterId1      uint32
		characterId2      uint32
		scheduledAt       time.Time
		invitees          []uint32
		tenantId          uuid.UUID
		mockExpectations  func(sqlmock.Sqlmock)
		expectedError     bool
		expectedMarriage  uint32
		expectedChar1     uint32
		expectedChar2     uint32
		expectedStatus    CeremonyStatus
	}{
		{
			name:         "successful ceremony creation",
			marriageId:   uint32(789),
			characterId1: uint32(1001),
			characterId2: uint32(1002),
			scheduledAt:  scheduledAt,
			invitees:     invitees,
			tenantId:     tenantId,
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "ceremonies"`)).
					WithArgs(
						uint32(789),              // marriage_id
						uint32(1001),             // character_id_1
						uint32(1002),             // character_id_2
						CeremonyStatusScheduled,  // status
						sqlmock.AnyArg(),         // scheduled_at
						sqlmock.AnyArg(),         // started_at (nil)
						sqlmock.AnyArg(),         // completed_at (nil)
						sqlmock.AnyArg(),         // cancelled_at (nil)
						sqlmock.AnyArg(),         // postponed_at (nil)
						sqlmock.AnyArg(),         // invitees JSON
						tenantId,                 // tenant_id
						sqlmock.AnyArg(),         // created_at
						sqlmock.AnyArg(),         // updated_at
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			expectedError:    false,
			expectedMarriage: uint32(789),
			expectedChar1:    uint32(1001),
			expectedChar2:    uint32(1002),
			expectedStatus:   CeremonyStatusScheduled,
		},
		{
			name:         "database error during creation",
			marriageId:   uint32(789),
			characterId1: uint32(1001),
			characterId2: uint32(1002),
			scheduledAt:  scheduledAt,
			invitees:     invitees,
			tenantId:     tenantId,
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "ceremonies"`)).
					WithArgs(
						uint32(789),
						uint32(1001),
						uint32(1002),
						CeremonyStatusScheduled,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						tenantId,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel)

			// Execute the function
			provider := CreateCeremony(gormDB, log)(tt.marriageId, tt.characterId1, tt.characterId2, tt.scheduledAt, tt.invitees, tt.tenantId)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMarriage, result.MarriageId)
				assert.Equal(t, tt.expectedChar1, result.CharacterId1)
				assert.Equal(t, tt.expectedChar2, result.CharacterId2)
				assert.Equal(t, tt.expectedStatus, result.Status)
				assert.Equal(t, tt.tenantId, result.TenantId)
				assert.False(t, result.ScheduledAt.IsZero())
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestUpdateCeremony(t *testing.T) {
	tenantId := uuid.New()
	
	tests := []struct {
		name             string
		ceremonyId       uint32
		entity           CeremonyEntity
		tenantId         uuid.UUID
		mockExpectations func(sqlmock.Sqlmock)
		expectedError    bool
	}{
		{
			name:       "successful ceremony update",
			ceremonyId: uint32(123),
			entity: CeremonyEntity{
				ID:           uint32(123),
				MarriageId:   uint32(789),
				CharacterId1: uint32(1001),
				CharacterId2: uint32(1002),
				Status:       CeremonyStatusActive,
				ScheduledAt:  time.Now().Add(-time.Hour),
				StartedAt:    &time.Time{},
				TenantId:     tenantId,
				CreatedAt:    time.Now().Add(-2*time.Hour),
				UpdatedAt:    time.Now().Add(-time.Hour),
			},
			tenantId: tenantId,
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "ceremonies" SET`)).
					WithArgs(
						uint32(789),              // marriage_id
						uint32(1001),             // character_id_1
						uint32(1002),             // character_id_2
						CeremonyStatusActive,     // status
						sqlmock.AnyArg(),         // scheduled_at
						sqlmock.AnyArg(),         // started_at
						sqlmock.AnyArg(),         // completed_at
						sqlmock.AnyArg(),         // cancelled_at
						sqlmock.AnyArg(),         // postponed_at
						sqlmock.AnyArg(),         // invitees
						tenantId,                 // tenant_id
						sqlmock.AnyArg(),         // created_at
						sqlmock.AnyArg(),         // updated_at
						uint32(123),              // id
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:       "database error during update",
			ceremonyId: uint32(123),
			entity: CeremonyEntity{
				ID:           uint32(123),
				MarriageId:   uint32(789),
				CharacterId1: uint32(1001),
				CharacterId2: uint32(1002),
				Status:       CeremonyStatusCancelled,
				TenantId:     tenantId,
			},
			tenantId: tenantId,
			mockExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "ceremonies" SET`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						uint32(123),
					).
					WillReturnError(gorm.ErrInvalidTransaction)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Configure GORM with mock
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: mockDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			require.NoError(t, err)

			// Setup expectations
			tt.mockExpectations(mock)

			// Create logger
			log := logrus.New()
			log.SetLevel(logrus.FatalLevel)

			// Execute the function
			provider := UpdateCeremony(gormDB, log)(tt.ceremonyId, tt.entity, tt.tenantId)
			result, err := provider()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.entity.ID, result.ID)
				assert.Equal(t, tt.entity.MarriageId, result.MarriageId)
				assert.Equal(t, tt.entity.Status, result.Status)
				// Updated timestamp should be updated
				assert.True(t, result.UpdatedAt.After(tt.entity.UpdatedAt) || result.UpdatedAt.Equal(tt.entity.UpdatedAt))
			}

			// Ensure all expectations were met (only for successful cases)
			if !tt.expectedError {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

// Test helper functions and edge cases
func TestAdministratorFunctionPatterns(t *testing.T) {
	t.Run("curried function pattern", func(t *testing.T) {
		// Test that all administrator functions follow the curried pattern
		// by checking that they return functions when called with dependencies
		
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err)

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		// Test CreateProposal currying
		createProposalFunc := CreateProposal(gormDB, log)
		assert.NotNil(t, createProposalFunc)
		
		proposalProvider := createProposalFunc(uint32(1), uint32(2), uuid.New())
		assert.NotNil(t, proposalProvider)

		// Test UpdateProposal currying
		updateProposalFunc := UpdateProposal(gormDB, log)
		assert.NotNil(t, updateProposalFunc)

		// Test CreateMarriage currying
		createMarriageFunc := CreateMarriage(gormDB, log)
		assert.NotNil(t, createMarriageFunc)
		
		marriageProvider := createMarriageFunc(uint32(1), uint32(2), uuid.New())
		assert.NotNil(t, marriageProvider)

		// Test UpdateMarriage currying
		updateMarriageFunc := UpdateMarriage(gormDB, log)
		assert.NotNil(t, updateMarriageFunc)

		// Test CreateCeremony currying
		createCeremonyFunc := CreateCeremony(gormDB, log)
		assert.NotNil(t, createCeremonyFunc)
		
		ceremonyProvider := createCeremonyFunc(uint32(1), uint32(2), uint32(3), time.Now(), []uint32{}, uuid.New())
		assert.NotNil(t, ceremonyProvider)

		// Test UpdateCeremony currying
		updateCeremonyFunc := UpdateCeremony(gormDB, log)
		assert.NotNil(t, updateCeremonyFunc)
	})

	t.Run("provider pattern compliance", func(t *testing.T) {
		// Test that all functions return proper Provider[T] types
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err)

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		tenantId := uuid.New()

		// All providers should be callable (even if they might fail due to no mock expectations)
		proposalProvider := CreateProposal(gormDB, log)(uint32(1), uint32(2), tenantId)
		_, err = proposalProvider()
		// Error is expected since we have no mock expectations, but function should be callable
		assert.Error(t, err)

		marriageProvider := CreateMarriage(gormDB, log)(uint32(1), uint32(2), tenantId)
		_, err = marriageProvider()
		assert.Error(t, err)

		ceremonyProvider := CreateCeremony(gormDB, log)(uint32(1), uint32(2), uint32(3), time.Now(), []uint32{}, tenantId)
		_, err = ceremonyProvider()
		assert.Error(t, err)
	})
}

// Custom AnyTime matcher for time values
type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}