package pending_change

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// newTestDB starts a disposable Postgres container and returns a connected
// *gorm.DB. The partial unique indexes this package's migration creates
// (WHERE status = 'PENDING' [AND type = 'NAME_CHANGE']) are the mechanism
// behind FR-2.3 and FR-3.3; SQLite's partial-index support does not reject
// these violations the same way Postgres does, so these tests need the real
// engine, mirroring libs/atlas-outbox/notify_test.go's container setup.
//
// database.RegisterTenantCallbacks is registered here to mirror what
// database.Connect does in production, matching the sibling packages'
// test-DB setup (e.g. teleport_rock's testDatabase). This package's own
// queries (provider.go/administrator.go) additionally filter on an explicit
// tenant_id parameter rather than relying on the callback's context-derived
// injection (see TestReadsAreTenantScoped) — that explicit predicate is what
// this package's correctness actually depends on and is tested for, so the
// callback registration here is defense-in-depth, not a behavior change.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)
	return db
}

func TestMigrationCreatesPartialUniqueIndexes(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration is not idempotent: %v", err)
	}

	tid := uuid.New()
	name := "Alpha"
	lower := "alpha"
	mk := func(charId uint32, status string) *entity {
		return &entity{
			Id: uuid.New(), TenantId: tid, CharacterId: charId,
			Type: TypeNameChange, Status: status,
			RequestedName: &name, RequestedNameLower: &lower,
			SourceWorldId: world.Id(0),
			CreatedAt:     time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
	}

	if err := db.Create(mk(1, StatusPending)).Error; err != nil {
		t.Fatalf("first pending insert: %v", err)
	}
	// Same name, different character, still PENDING -> reservation index must reject.
	if err := db.Create(mk(2, StatusPending)).Error; err == nil {
		t.Fatal("expected reservation unique violation for a duplicate pending name")
	}
	// Same name once the first is terminal -> allowed.
	if err := db.Model(&entity{}).Where("character_id = ?", uint32(1)).
		Update("status", StatusCancelled).Error; err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := db.Create(mk(2, StatusPending)).Error; err != nil {
		t.Fatalf("expected reservation released after terminal transition: %v", err)
	}
}

func TestMigrationEnforcesOnePendingPerType(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()
	wid := world.Id(1)
	mk := func() *entity {
		return &entity{
			Id: uuid.New(), TenantId: tid, CharacterId: 7,
			Type: TypeWorldTransfer, Status: StatusPending,
			DestinationWorldId: &wid, SourceWorldId: world.Id(0),
			CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
	}
	if err := db.Create(mk()).Error; err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := db.Create(mk()).Error; err == nil {
		t.Fatal("expected one-pending-per-type unique violation")
	}
}
