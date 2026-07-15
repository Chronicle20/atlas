package test

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"atlas-gachapons/reward"
	"testing"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateGachaponProcessor creates a new gachapon processor for testing
func CreateGachaponProcessor(t *testing.T) (gachapon.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, gachapon.Migration)
	ctx := CreateTestContext()
	processor := gachapon.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateItemProcessor creates a new item processor for testing
func CreateItemProcessor(t *testing.T) (item.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, gachapon.Migration, item.Migration)
	ctx := CreateTestContext()
	processor := item.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateGlobalProcessor creates a new global processor for testing
func CreateGlobalProcessor(t *testing.T) (global.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, global.Migration)
	ctx := CreateTestContext()
	processor := global.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateRewardProcessor creates a new reward processor for testing
func CreateRewardProcessor(t *testing.T) (reward.Processor, *gorm.DB, func()) {
	logger := logrus.New()
	db := SetupTestDB(t, gachapon.Migration, item.Migration, global.Migration)
	ctx := CreateTestContext()
	processor := reward.NewProcessor(logger, ctx, db)

	cleanup := func() {
		CleanupTestDB(t, db)
	}

	return processor, db, cleanup
}

// CreateGachaponProcessorWithDB creates a gachapon processor with an existing database
func CreateGachaponProcessorWithDB(t *testing.T, db *gorm.DB) gachapon.Processor {
	logger := logrus.New()
	ctx := CreateTestContext()
	return gachapon.NewProcessor(logger, ctx, db)
}

// CreateItemProcessorWithDB creates an item processor with an existing database
func CreateItemProcessorWithDB(t *testing.T, db *gorm.DB) item.Processor {
	logger := logrus.New()
	ctx := CreateTestContext()
	return item.NewProcessor(logger, ctx, db)
}

// CreateGlobalProcessorWithDB creates a global processor with an existing database
func CreateGlobalProcessorWithDB(t *testing.T, db *gorm.DB) global.Processor {
	logger := logrus.New()
	ctx := CreateTestContext()
	return global.NewProcessor(logger, ctx, db)
}
