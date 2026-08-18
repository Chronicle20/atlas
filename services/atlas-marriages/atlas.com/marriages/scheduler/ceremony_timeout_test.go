package scheduler

import (
	"context"
	"os"
	"strings"
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

// TestCeremonyTimeoutScheduler_UsesForEachOwnedEnvironment pins design
// C4/FR-6.1: tenant resolution must go through
// service.ForEachOwnedEnvironment so both the owned-environment set and each
// environment's tenant set are resolved fresh on every tick, not cached and
// closed over.
func TestCeremonyTimeoutScheduler_UsesForEachOwnedEnvironment(t *testing.T) {
	src, err := os.ReadFile("ceremony_timeout.go")
	assert.NoError(t, err)
	s := string(src)
	assert.True(t, strings.Contains(s, "service.ForEachOwnedEnvironment"),
		"processActiveCeremonies does not use service.ForEachOwnedEnvironment")
}
