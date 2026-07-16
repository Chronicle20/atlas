package test

import (
	"atlas-mts/bid"
	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/wish"
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

// CreateHoldingProcessor creates a new holding processor for testing, backed by
// an in-memory SQLite database migrated with the holding schema.
func CreateHoldingProcessor(t *testing.T) (holding.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, holding.Migration)
	ctx := CreateTestContext()
	processor := holding.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateBidProcessor creates a new bid processor for testing, backed by an
// in-memory SQLite database migrated with the bid schema.
func CreateBidProcessor(t *testing.T) (bid.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, bid.Migration)
	ctx := CreateTestContext()
	processor := bid.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateWishProcessor creates a new wish processor for testing, backed by an
// in-memory SQLite database migrated with the wish schema.
func CreateWishProcessor(t *testing.T) (wish.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, wish.Migration)
	ctx := CreateTestContext()
	processor := wish.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}
