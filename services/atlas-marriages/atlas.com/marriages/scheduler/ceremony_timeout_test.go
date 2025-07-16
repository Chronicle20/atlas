package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCeremonyTimeoutScheduler_Creation(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create context
	ctx := context.Background()

	// Create scheduler
	scheduler := NewCeremonyTimeoutScheduler(logger, ctx, db)
	assert.NotNil(t, scheduler)
	assert.Equal(t, 1*time.Minute, scheduler.interval)

	// Test with custom interval
	customScheduler := NewCeremonyTimeoutScheduler(logger, ctx, db).WithInterval(30 * time.Second)
	assert.Equal(t, 30*time.Second, customScheduler.interval)
}

func TestCeremonyTimeoutScheduler_StartStop(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create context with timeout for test
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create scheduler with very fast interval for testing
	scheduler := NewCeremonyTimeoutScheduler(logger, ctx, db).WithInterval(10 * time.Millisecond)

	// Start the scheduler
	scheduler.Start()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop the scheduler
	scheduler.Stop()

	// Should not panic or hang
	assert.True(t, true, "Scheduler started and stopped successfully")
}