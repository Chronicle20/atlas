package test

import (
	"atlas-npc/commodities"
	"atlas-npc/shops"
	"testing"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateCommoditiesProcessor creates a new commodities processor for testing
func CreateCommoditiesProcessor(t *testing.T) (commodities.Processor, *gorm.DB, func()) {
	// Set up logger
	logger := logrus.New()

	// Set up test database with migrations
	db := SetupTestDB(t, commodities.Migration)

	// Create test context
	ctx := CreateTestContext()

	// Create processor
	processor := commodities.NewProcessor(logger, ctx, db)

	// Return cleanup function
	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateShopsProcessor creates a new shops processor for testing
func CreateShopsProcessor(t *testing.T) (shops.Processor, *gorm.DB, func()) {
	// Set up logger
	logger := logrus.New()

	// Set up test database with migrations
	db := SetupTestDB(t, commodities.Migration, shops.Migration)

	// Create test context
	ctx := CreateTestContext()

	// Create processor
	processor := shops.NewProcessor(logger, ctx, db)

	// Return cleanup function
	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}
