package teleport_rock

import (
	"fmt"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	// Uniquely-named shared-cache in-memory database, mirroring
	// character/processor_test.go: a bare ":memory:" DB is private to one
	// connection, so a second pooled connection can see an empty schema.
	// Shared-cache is visible to every pooled connection; the unique name
	// keeps each test isolated.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetConnMaxIdleTime(0)
	}

	database.RegisterTenantCallbacks(l, db)
	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func TestReplaceListAndGet(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()

	if err := replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{100000000, 220000000}); err != nil {
		t.Fatalf("replaceList: %v", err)
	}
	if err := replaceList(db, tenantId, 42, ListTypeVip, []_map.Id{104040000}); err != nil {
		t.Fatalf("replaceList vip: %v", err)
	}

	es, err := getByCharacterId(db, tenantId, 42)
	if err != nil {
		t.Fatalf("getByCharacterId: %v", err)
	}
	if len(es) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(es))
	}

	// Replace compacts: overwriting the regular list removes stale rows.
	if err := replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{220000000}); err != nil {
		t.Fatalf("replaceList overwrite: %v", err)
	}
	es, _ = getByCharacterId(db, tenantId, 42)
	regular := 0
	for _, e := range es {
		if e.ListType == ListTypeRegular {
			if e.Slot != 0 || e.MapId != 220000000 {
				t.Fatalf("expected compacted slot 0 map 220000000, got slot %d map %d", e.Slot, e.MapId)
			}
			regular++
		}
	}
	if regular != 1 {
		t.Fatalf("expected 1 regular row, got %d", regular)
	}
}

func TestTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	a, b := uuid.New(), uuid.New()
	_ = replaceList(db, a, 42, ListTypeRegular, []_map.Id{100000000})
	es, err := getByCharacterId(db, b, 42)
	if err != nil {
		t.Fatalf("getByCharacterId: %v", err)
	}
	if len(es) != 0 {
		t.Fatalf("tenant b must see no rows, got %d", len(es))
	}
}

func TestDeleteForCharacter(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	_ = replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{100000000})
	_ = replaceList(db, tenantId, 42, ListTypeVip, []_map.Id{104040000})
	if err := DeleteForCharacter(db, tenantId, 42); err != nil {
		t.Fatalf("DeleteForCharacter: %v", err)
	}
	es, _ := getByCharacterId(db, tenantId, 42)
	if len(es) != 0 {
		t.Fatalf("expected 0 rows after delete, got %d", len(es))
	}
}
