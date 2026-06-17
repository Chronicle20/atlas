package test

import (
	"atlas-mts/listing"
	"testing"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateListingProcessor creates a new listing processor for testing, backed by
// an in-memory SQLite database migrated with the listing schema.
func CreateListingProcessor(t *testing.T) (listing.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, listing.Migration)
	ctx := CreateTestContext()
	processor := listing.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}
