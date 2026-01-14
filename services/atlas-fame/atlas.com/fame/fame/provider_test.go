package fame

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func createTestEntity(db *gorm.DB, tenantId uuid.UUID, characterId uint32, targetId uint32, amount int8, createdAt time.Time) Entity {
	e := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		TargetId:    targetId,
		Amount:      amount,
		CreatedAt:   createdAt,
	}
	db.Create(&e)
	return e
}

func TestByCharacterIdLastMonthEntityProvider_ReturnsMatchingEntities(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId := uint32(1000)

	// Create entity within last month
	now := time.Now()
	createTestEntity(db, tenantId, characterId, 2000, 1, now.AddDate(0, 0, -5))

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, characterId, result[0].CharacterId)
}

func TestByCharacterIdLastMonthEntityProvider_ReturnsMultipleEntities(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId := uint32(1000)

	now := time.Now()
	createTestEntity(db, tenantId, characterId, 2000, 1, now.AddDate(0, 0, -5))
	createTestEntity(db, tenantId, characterId, 2001, 1, now.AddDate(0, 0, -10))
	createTestEntity(db, tenantId, characterId, 2002, -1, now.AddDate(0, 0, -15))

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestByCharacterIdLastMonthEntityProvider_ExcludesOldEntities(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId := uint32(1000)

	now := time.Now()
	// Entity within last month - should be included
	createTestEntity(db, tenantId, characterId, 2000, 1, now.AddDate(0, 0, -5))
	// Entity older than last month - should be excluded
	createTestEntity(db, tenantId, characterId, 2001, 1, now.AddDate(0, -2, 0))

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(2000), result[0].TargetId)
}

func TestByCharacterIdLastMonthEntityProvider_FiltersByTenant(t *testing.T) {
	db := testDatabase(t)
	tenantId1 := uuid.New()
	tenantId2 := uuid.New()
	characterId := uint32(1000)

	now := time.Now()
	createTestEntity(db, tenantId1, characterId, 2000, 1, now.AddDate(0, 0, -5))
	createTestEntity(db, tenantId2, characterId, 2001, 1, now.AddDate(0, 0, -5))

	provider := byCharacterIdLastMonthEntityProvider(tenantId1, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, tenantId1, result[0].TenantId)
}

func TestByCharacterIdLastMonthEntityProvider_FiltersByCharacterId(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId1 := uint32(1000)
	characterId2 := uint32(1001)

	now := time.Now()
	createTestEntity(db, tenantId, characterId1, 2000, 1, now.AddDate(0, 0, -5))
	createTestEntity(db, tenantId, characterId2, 2001, 1, now.AddDate(0, 0, -5))

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId1)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, characterId1, result[0].CharacterId)
}

func TestByCharacterIdLastMonthEntityProvider_ReturnsEmptyForNoMatches(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId := uint32(1000)

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestByCharacterIdLastMonthEntityProvider_BoundaryDateExactlyOneMonth(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	characterId := uint32(1000)

	// Create entity exactly at the boundary (should be included)
	lastMonth := time.Now().AddDate(0, -1, 0)
	createTestEntity(db, tenantId, characterId, 2000, 1, lastMonth.Add(time.Hour))

	provider := byCharacterIdLastMonthEntityProvider(tenantId, characterId)
	result, err := provider(db)()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}
