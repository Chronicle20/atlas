package marriage

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCeremonyStateTransitions tests basic ceremony state transitions
func TestCeremonyStateTransitions(t *testing.T) {
	// Set up in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	
	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	assert.NoError(t, err)
	
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create tenant context
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	assert.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	// Create processor (skip character processor for this test)
	processor := &ProcessorImpl{
		log: logger,
		ctx: ctx,
		db:  db,
	}
	
	// Create a test marriage first
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
	assert.NoError(t, err)
	
	// Test scheduling a ceremony
	scheduledAt := now.Add(time.Hour)
	invitees := []uint32{3, 4, 5}
	
	ceremony, err := processor.ScheduleCeremony(marriageEntity.ID, scheduledAt, invitees)()
	assert.NoError(t, err)
	assert.Equal(t, CeremonyStatusScheduled, ceremony.Status())
	assert.Equal(t, marriageEntity.ID, ceremony.MarriageId())
	assert.Equal(t, marriageEntity.CharacterId1, ceremony.CharacterId1())
	assert.Equal(t, marriageEntity.CharacterId2, ceremony.CharacterId2())
	assert.Equal(t, invitees, ceremony.Invitees())
	
	// Test starting the ceremony
	startedCeremony, err := processor.StartCeremony(ceremony.Id())()
	assert.NoError(t, err)
	assert.Equal(t, CeremonyStatusActive, startedCeremony.Status())
	assert.NotNil(t, startedCeremony.StartedAt())
	
	// Test completing the ceremony
	completedCeremony, err := processor.CompleteCeremony(ceremony.Id())()
	assert.NoError(t, err)
	assert.Equal(t, CeremonyStatusCompleted, completedCeremony.Status())
	assert.NotNil(t, completedCeremony.CompletedAt())
}

// TestCeremonyInviteeManagement tests adding and removing invitees
func TestCeremonyInviteeManagement(t *testing.T) {
	// Set up in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	
	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	assert.NoError(t, err)
	
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create tenant context
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	assert.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	// Create processor
	processor := &ProcessorImpl{
		log: logger,
		ctx: ctx,
		db:  db,
	}
	
	// Create a test marriage and ceremony
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
	assert.NoError(t, err)
	
	// Schedule ceremony with initial invitees
	scheduledAt := now.Add(time.Hour)
	initialInvitees := []uint32{3, 4}
	
	ceremony, err := processor.ScheduleCeremony(marriageEntity.ID, scheduledAt, initialInvitees)()
	assert.NoError(t, err)
	assert.Equal(t, 2, ceremony.InviteeCount())
	
	// Add a new invitee
	updatedCeremony, err := processor.AddInvitee(ceremony.Id(), 5)()
	assert.NoError(t, err)
	assert.Equal(t, 3, updatedCeremony.InviteeCount())
	assert.True(t, updatedCeremony.IsInvited(5))
	
	// Remove an invitee
	finalCeremony, err := processor.RemoveInvitee(ceremony.Id(), 3)()
	assert.NoError(t, err)
	assert.Equal(t, 2, finalCeremony.InviteeCount())
	assert.False(t, finalCeremony.IsInvited(3))
	assert.True(t, finalCeremony.IsInvited(4))
	assert.True(t, finalCeremony.IsInvited(5))
}

// TestCeremonyQueries tests ceremony query methods
func TestCeremonyQueries(t *testing.T) {
	// Set up in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	
	// Run migrations
	err = db.AutoMigrate(&Entity{}, &ProposalEntity{}, &CeremonyEntity{})
	assert.NoError(t, err)
	
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create tenant context
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	assert.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	// Create processor
	processor := &ProcessorImpl{
		log: logger,
		ctx: ctx,
		db:  db,
	}
	
	// Create test ceremonies
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
	assert.NoError(t, err)
	
	// Schedule ceremony
	scheduledAt := now.Add(time.Hour)
	ceremony, err := processor.ScheduleCeremony(marriageEntity.ID, scheduledAt, []uint32{})()
	assert.NoError(t, err)
	
	// Test GetCeremonyById
	retrievedCeremony, err := processor.GetCeremonyById(ceremony.Id())()
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCeremony)
	assert.Equal(t, ceremony.Id(), retrievedCeremony.Id())
	
	// Test GetCeremonyByMarriage
	marriageCeremony, err := processor.GetCeremonyByMarriage(marriageEntity.ID)()
	assert.NoError(t, err)
	assert.NotNil(t, marriageCeremony)
	assert.Equal(t, ceremony.Id(), marriageCeremony.Id())
	
	// Test GetUpcomingCeremonies
	upcomingCeremonies, err := processor.GetUpcomingCeremonies()()
	assert.NoError(t, err)
	assert.Len(t, upcomingCeremonies, 1)
	assert.Equal(t, ceremony.Id(), upcomingCeremonies[0].Id())
	
	// Start ceremony and test GetActiveCeremonies
	_, err = processor.StartCeremony(ceremony.Id())()
	assert.NoError(t, err)
	
	activeCeremonies, err := processor.GetActiveCeremonies()()
	assert.NoError(t, err)
	assert.Len(t, activeCeremonies, 1)
	assert.Equal(t, ceremony.Id(), activeCeremonies[0].Id())
}