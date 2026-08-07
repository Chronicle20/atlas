package report

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err = db.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
	return db
}

func sampleTenant() tenant.Model {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tm
}

func testContext(tm tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), tm)
}

func TestCreateAndFetchReport(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	chatLog := "bob: rude things"
	m, err := create(tdb)(tm.Id(), KindClaim, 1, "Reporter", 2, "Accused", 3, "harassment", &chatLog,
		[]TranscriptLine{{Timestamp: 10, SenderId: 2, SenderName: "Accused", ChatType: "GENERAL", Text: "rude things"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.Id() == uuid.Nil {
		t.Fatal("expected generated id")
	}

	e, err := entityById(m.Id())(tdb)()
	if err != nil {
		t.Fatalf("entityById: %v", err)
	}
	got, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if got.Kind() != KindClaim || got.Status() != StatusOpen || got.AccusedName() != "Accused" {
		t.Errorf("fetched mismatch: %+v", got)
	}
	if got.ChatLog() == nil || *got.ChatLog() != chatLog {
		t.Error("chat log not persisted")
	}
	if len(got.ServerTranscript()) != 1 || got.ServerTranscript()[0].Text != "rude things" {
		t.Errorf("transcript not round-tripped: %+v", got.ServerTranscript())
	}
}

func TestCreateNilTranscriptStaysNil(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m, err := create(tdb)(tm.Id(), KindSue, 1, "Reporter", 2, "Accused", 0, "spamming", nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	e, err := entityById(m.Id())(tdb)()
	if err != nil {
		t.Fatalf("entityById: %v", err)
	}
	got, _ := Make(e)
	if got.ChatLog() != nil || got.ServerTranscript() != nil {
		t.Errorf("expected nil chat log and transcript, got %+v / %+v", got.ChatLog(), got.ServerTranscript())
	}
}

func TestUpdateStatus(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m, _ := create(tdb)(tm.Id(), KindSue, 1, "Reporter", 2, "Accused", 0, "spamming", nil, nil)
	if err := updateStatus(tdb)(m.Id(), StatusReviewed); err != nil {
		t.Fatalf("updateStatus: %v", err)
	}
	e, _ := entityById(m.Id())(tdb)()
	if e.Status != string(StatusReviewed) {
		t.Errorf("status: got %s want %s", e.Status, StatusReviewed)
	}
	if err := updateStatus(tdb)(uuid.New(), StatusReviewed); err == nil {
		t.Error("expected error for unknown id")
	}
}

func TestEntitiesByStatusFilters(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m1, _ := create(tdb)(tm.Id(), KindSue, 1, "R", 2, "A", 0, "one", nil, nil)
	_, _ = create(tdb)(tm.Id(), KindSue, 1, "R", 3, "B", 0, "two", nil, nil)
	_ = updateStatus(tdb)(m1.Id(), StatusActioned)

	open, err := entitiesByStatus(StatusOpen)(tdb)()
	if err != nil {
		t.Fatalf("entitiesByStatus: %v", err)
	}
	if len(open) != 1 || open[0].Description != "two" {
		t.Errorf("open filter mismatch: %+v", open)
	}
	all, _ := entitiesByTenant()(tdb)()
	if len(all) != 2 {
		t.Errorf("expected 2 rows, got %d", len(all))
	}
}

func TestEntityByIdFiltersByTenant(t *testing.T) {
	db := setupTestDatabase(t)
	tmA := sampleTenant()
	tmB := sampleTenant()

	m, err := create(db.WithContext(testContext(tmA)))(tmA.Id(), KindSue, 1, "R", 2, "A", 0, "tenant A's report", nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Tenant A can fetch its own report by id.
	gotA, err := entityById(m.Id())(db.WithContext(testContext(tmA)))()
	if err != nil {
		t.Fatalf("entityById (tenant A): %v", err)
	}
	if gotA.TenantId != tmA.Id() {
		t.Errorf("expected tenant A's row, got tenant %s", gotA.TenantId)
	}

	// Tenant B must not see tenant A's row by id.
	_, err = entityById(m.Id())(db.WithContext(testContext(tmB)))()
	if err == nil {
		t.Error("expected error: tenant B must not see tenant A's row by id")
	}
}

func TestEntitiesByTenantIsolatedPerTenant(t *testing.T) {
	db := setupTestDatabase(t)
	tmA := sampleTenant()
	tmB := sampleTenant()
	tdbA := db.WithContext(testContext(tmA))
	tdbB := db.WithContext(testContext(tmB))

	m1, _ := create(tdbA)(tmA.Id(), KindSue, 1, "R", 2, "A1", 0, "a-one", nil, nil)
	_, _ = create(tdbA)(tmA.Id(), KindSue, 1, "R", 3, "A2", 0, "a-two", nil, nil)
	_, _ = create(tdbB)(tmB.Id(), KindSue, 1, "R", 4, "B1", 0, "b-one", nil, nil)

	allA, err := entitiesByTenant()(tdbA)()
	if err != nil {
		t.Fatalf("entitiesByTenant (tenant A): %v", err)
	}
	if len(allA) != 2 {
		t.Errorf("tenant isolation failed: expected 2 reports for tenant A, got %d", len(allA))
	}

	allB, err := entitiesByTenant()(tdbB)()
	if err != nil {
		t.Fatalf("entitiesByTenant (tenant B): %v", err)
	}
	if len(allB) != 1 {
		t.Errorf("tenant isolation failed: expected 1 report for tenant B, got %d", len(allB))
	}

	// entitiesByStatus must also be tenant-scoped: mark one of tenant A's
	// reports actioned and confirm tenant B's open-status view is unaffected.
	_ = updateStatus(tdbA)(m1.Id(), StatusActioned)

	openA, err := entitiesByStatus(StatusOpen)(tdbA)()
	if err != nil {
		t.Fatalf("entitiesByStatus (tenant A): %v", err)
	}
	if len(openA) != 1 || openA[0].AccusedName != "A2" {
		t.Errorf("tenant A open filter mismatch: %+v", openA)
	}

	openB, err := entitiesByStatus(StatusOpen)(tdbB)()
	if err != nil {
		t.Fatalf("entitiesByStatus (tenant B): %v", err)
	}
	if len(openB) != 1 || openB[0].AccusedName != "B1" {
		t.Errorf("tenant B open filter mismatch: %+v", openB)
	}
}
