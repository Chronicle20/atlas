package area_info_test

import (
	"atlas-character/area_info"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	if err := area_info.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testProcessor(t *testing.T, tctx context.Context, db *gorm.DB) area_info.Processor {
	return area_info.NewProcessor(testLogger(), tctx, db)
}

func TestUpsertAreaInfoReplacesWholeString(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(t, tctx, db)

	m1 := area_info.NewBuilder().SetCharacterId(1).SetArea(21019).SetInfo("miss=o;helper=clear").Build()
	if _, err := p.Put(m1); err != nil {
		t.Fatalf("Put: %v", err)
	}

	m2 := area_info.NewBuilder().SetCharacterId(1).SetArea(21019).SetInfo("miss=o;arr=o;helper=clear").Build()
	if _, err := p.Put(m2); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := p.GetByAreaAsString(1, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString: %v", err)
	}
	if got != "miss=o;arr=o;helper=clear" {
		t.Fatalf("expected a replace, got %q", got)
	}
}

func TestGetByAreaMissingReturnsEmpty(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(t, tctx, db)

	got, err := p.GetByAreaAsString(1, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string for an unset area, got %q", got)
	}
}

func TestAreaInfoIsPerCharacter(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := testProcessor(t, tctx, db)

	if _, err := p.Put(area_info.NewBuilder().SetCharacterId(1).SetArea(21019).SetInfo("char1").Build()); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := p.Put(area_info.NewBuilder().SetCharacterId(2).SetArea(21019).SetInfo("char2").Build()); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got1, err := p.GetByAreaAsString(1, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString(1): %v", err)
	}
	got2, err := p.GetByAreaAsString(2, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString(2): %v", err)
	}
	if got1 != "char1" || got2 != "char2" {
		t.Fatalf("expected independent per-character values, got %q and %q", got1, got2)
	}
}

func TestAreaInfoIsTenantScoped(t *testing.T) {
	db := testDatabase(t)

	tctxA := tenant.WithContext(context.Background(), testTenant())
	pA := testProcessor(t, tctxA, db)
	if _, err := pA.Put(area_info.NewBuilder().SetCharacterId(1).SetArea(21019).SetInfo("tenantA").Build()); err != nil {
		t.Fatalf("Put: %v", err)
	}

	tctxB := tenant.WithContext(context.Background(), testTenant())
	pB := testProcessor(t, tctxB, db)

	gotB, err := pB.GetByAreaAsString(1, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString: %v", err)
	}
	if gotB != "" {
		t.Fatalf("expected tenant B to see no value, got %q", gotB)
	}

	gotA, err := pA.GetByAreaAsString(1, 21019)
	if err != nil {
		t.Fatalf("GetByAreaAsString: %v", err)
	}
	if gotA != "tenantA" {
		t.Fatalf("expected tenant A to still see its own value, got %q", gotA)
	}
}
