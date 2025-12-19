package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	return db
}

func TestNewProposalExpiryScheduler(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db)
	
	if scheduler == nil {
		t.Error("Expected scheduler to be created, got nil")
	}
}

func TestProposalExpiryScheduler_WithInterval(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db)
	interval := 30 * time.Second
	
	updatedScheduler := scheduler.WithInterval(interval)
	
	if updatedScheduler == nil {
		t.Error("Expected scheduler to be returned, got nil")
	}
}

func TestProposalExpiryScheduler_StartStop(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db).WithInterval(50 * time.Millisecond)
	
	// Start the scheduler
	scheduler.Start()
	
	// Let it run for a short time
	time.Sleep(200 * time.Millisecond)
	
	// Stop the scheduler
	scheduler.Stop()
	
	// Test should complete without hanging
}

func TestProposalExpiryScheduler_Run(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db).WithInterval(50 * time.Millisecond)
	
	// This should run for the timeout duration and then stop
	scheduler.run()
}

func TestProposalExpiryScheduler_ProcessExpiredProposals(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db)
	
	// Test processing expired proposals
	scheduler.processExpiredProposals()
}

func TestProposalExpiryScheduler_GetTenantsWithProposals(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db)
	
	// Test getting tenants with proposals
	tenants, err := scheduler.getTenantsWithProposals()
	// The table doesn't exist, so we expect an error
	if err == nil {
		t.Error("Expected error due to missing table, got none")
	}
	
	// Should return empty list for error case
	if len(tenants) != 0 {
		t.Errorf("Expected empty tenants list, got %d", len(tenants))
	}
}

func TestProposalExpiryScheduler_ProcessExpiredProposalsForTenant(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New()
	ctx := context.Background()
	
	scheduler := NewProposalExpiryScheduler(log, ctx, db)
	
	// Create a test tenant
	tenantId := uuid.New()
	
	// Test processing expired proposals for specific tenant
	scheduler.processExpiredProposalsForTenant(tenantId)
}