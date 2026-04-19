package _map

import (
	"atlas-data/map/npc"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// test-compatible mirrors of production entities (sqlite-safe types).

type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (testDocumentEntity) TableName() string { return "documents" }

type testSearchIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testSearchIndexEntity) TableName() string { return "map_search_index" }

type testMonsterSpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	MonsterId  uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testMonsterSpawnIndexEntity) TableName() string { return "monster_spawn_index" }

type testNpcSpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	NpcId      uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testNpcSpawnIndexEntity) TableName() string { return "npc_spawn_index" }

func setupStorageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{SlowThreshold: time.Second, LogLevel: logger.Silent, Colorful: false},
		),
	})
	require.NoError(t, err)

	// start clean each test
	db.Migrator().DropTable(&testDocumentEntity{}, &testSearchIndexEntity{}, &testMonsterSpawnIndexEntity{}, &testNpcSpawnIndexEntity{})
	require.NoError(t, db.AutoMigrate(&testDocumentEntity{}, &testSearchIndexEntity{}, &testMonsterSpawnIndexEntity{}, &testNpcSpawnIndexEntity{}))

	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func newTestTenant(t *testing.T) tenant.Model {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

func TestStorage_Add_InsertsBothRows(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	m := RestModel{Id: 100000000, Name: "Henesys", StreetName: "Victoria Road"}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	var docCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ? AND document_id = ?", "MAP", 100000000).Count(&docCount).Error)
	assert.Equal(t, int64(1), docCount)

	var idx testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 100000000).First(&idx).Error)
	assert.Equal(t, "Henesys", idx.Name)
	assert.Equal(t, "Victoria Road", idx.StreetName)
	assert.Equal(t, tn.Id(), idx.TenantId)
}

func TestStorage_Add_RollbackOnIndexFailure(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	// Register a gorm callback that fails on inserts to map_search_index.
	err := db.Callback().Create().After("gorm:create").Register("test:fail_index", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "map_search_index" {
			tx.AddError(errors.New("forced index failure"))
		}
	})
	require.NoError(t, err)
	defer db.Callback().Create().Remove("test:fail_index")

	s := NewStorage(l, db)
	m := RestModel{Id: 100000000, Name: "Henesys", StreetName: "Victoria Road"}
	_, addErr := s.Add(ctx)(m)()
	require.Error(t, addErr)

	var docCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "MAP").Count(&docCount).Error)
	assert.Equal(t, int64(0), docCount, "documents insert should have rolled back on index failure")

	var idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), idxCount)
}

func TestStorage_Clear_EmptiesBothTables(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	_, err := s.Add(ctx)(RestModel{Id: 1, Name: "A", StreetName: "X"})()
	require.NoError(t, err)
	_, err = s.Add(ctx)(RestModel{Id: 2, Name: "B", StreetName: "Y"})()
	require.NoError(t, err)

	require.NoError(t, s.Clear(ctx))

	var docCount, idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "MAP").Count(&docCount).Error)
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), docCount)
	assert.Equal(t, int64(0), idxCount)
}

func TestStorage_Add_PopulatesNpcSpawnIndex(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	m := RestModel{
		Id:         100000000,
		Name:       "Henesys",
		StreetName: "Victoria Road",
		NPCs: []npc.RestModel{
			{Id: 1, Template: 9200000},
			{Id: 2, Template: 9200000},
			{Id: 3, Template: 2040000},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	var rows []testNpcSpawnIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 100000000).Find(&rows).Error)
	assert.Len(t, rows, 2)

	counts := make(map[uint32]uint32)
	for _, r := range rows {
		counts[r.NpcId] = r.SpawnCount
		assert.Equal(t, "Henesys", r.Name)
		assert.Equal(t, "Victoria Road", r.StreetName)
		assert.Equal(t, tn.Id(), r.TenantId)
	}
	assert.Equal(t, uint32(2), counts[9200000])
	assert.Equal(t, uint32(1), counts[2040000])
}

func TestStorage_Add_ReplacesNpcSpawnIndexRows(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	_, err := s.Add(ctx)(RestModel{
		Id: 100000000, Name: "Henesys", StreetName: "Victoria Road",
		NPCs: []npc.RestModel{{Id: 1, Template: 9200000}, {Id: 2, Template: 9200001}},
	})()
	require.NoError(t, err)

	_, err = s.Add(ctx)(RestModel{
		Id: 100000000, Name: "Henesys", StreetName: "Victoria Road",
		NPCs: []npc.RestModel{{Id: 1, Template: 9200000}},
	})()
	require.NoError(t, err)

	var rows []testNpcSpawnIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 100000000).Find(&rows).Error)
	assert.Len(t, rows, 1)
	assert.Equal(t, uint32(9200000), rows[0].NpcId)
}

func TestStorage_Clear_OtherDocTypeUntouched(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	_, err := s.Add(ctx)(RestModel{Id: 1, Name: "A", StreetName: "X"})()
	require.NoError(t, err)

	// seed a non-MAP document directly.
	npcDoc := testDocumentEntity{
		Id: uuid.New(), TenantId: tn.Id(), Type: "NPC", DocumentId: 2003, Content: json.RawMessage(`{"data":{}}`),
	}
	require.NoError(t, db.WithContext(ctx).Create(&npcDoc).Error)

	require.NoError(t, s.Clear(ctx))

	var npcCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "NPC").Count(&npcCount).Error)
	assert.Equal(t, int64(1), npcCount, "NPC document should be untouched")
}
