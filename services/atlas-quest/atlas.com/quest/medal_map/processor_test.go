package medal_map_test

import (
	"atlas-quest/medal_map"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	// Uniquely-named shared-cache in-memory database: a bare ":memory:" DB is
	// private to one connection, so a second pooled connection can see an
	// empty schema. Shared-cache is visible to every pooled connection; the
	// unique name keeps each test isolated.
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
	if err := medal_map.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tm
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testProcessor(tctx context.Context, db *gorm.DB) medal_map.Processor {
	return medal_map.NewProcessor(testLogger(), tctx, db)
}

func TestRecordMedalMapDeduplicates(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(tctx, db)

	first, err := p.Record(1, 29005, _map.Id(100000000))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if first.Count != 1 || !first.NewlyRecorded {
		t.Fatalf("first record = %+v, want {Count:1 NewlyRecorded:true}", first)
	}

	second, err := p.Record(1, 29005, _map.Id(100000000))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if second.Count != 1 || second.NewlyRecorded {
		t.Fatalf("second record = %+v, want {Count:1 NewlyRecorded:false}", second)
	}
}

func TestRecordMedalMapCountsDistinctMaps(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(tctx, db)

	maps := []_map.Id{100000000, 100000001, 100000000, 100000002}
	var last medal_map.RecordResult
	for _, mapId := range maps {
		result, err := p.Record(1, 29005, mapId)
		if err != nil {
			t.Fatalf("Record(%d): %v", mapId, err)
		}
		last = result
	}
	if last.Count != 3 {
		t.Fatalf("final count = %d, want 3", last.Count)
	}
}

func TestMedalMapsArePerQuest(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(tctx, db)

	if _, err := p.Record(1, 29005, _map.Id(100000000)); err != nil {
		t.Fatalf("Record: %v", err)
	}

	other, err := p.Record(1, 29006, _map.Id(100000001))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if other.Count != 1 {
		t.Fatalf("quest 29006 count = %d, want 1 (unaffected by quest 29005's recording)", other.Count)
	}
}

func TestMedalMapsArePerCharacterAndTenant(t *testing.T) {
	db := testDatabase(t)

	tctxA := tenant.WithContext(context.Background(), testTenant())
	pA := testProcessor(tctxA, db)
	if _, err := pA.Record(1, 29005, _map.Id(100000000)); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := pA.Record(1, 29005, _map.Id(100000001)); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Different character, same tenant: independent count.
	otherChar, err := pA.Record(2, 29005, _map.Id(100000000))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if otherChar.Count != 1 {
		t.Fatalf("character 2 count = %d, want 1", otherChar.Count)
	}

	// Different tenant, same character/quest/map: independent count.
	tctxB := tenant.WithContext(context.Background(), testTenant())
	pB := testProcessor(tctxB, db)
	otherTenant, err := pB.Record(1, 29005, _map.Id(100000000))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if otherTenant.Count != 1 || !otherTenant.NewlyRecorded {
		t.Fatalf("tenant B record = %+v, want {Count:1 NewlyRecorded:true}", otherTenant)
	}

	// Tenant A's count is unaffected by tenant B's recording.
	stillA, err := pA.Record(1, 29005, _map.Id(100000002))
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if stillA.Count != 3 {
		t.Fatalf("tenant A count after third map = %d, want 3", stillA.Count)
	}
}
