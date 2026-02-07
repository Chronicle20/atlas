package history

import (
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	err = db.AutoMigrate(&Entity{})
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
	return db
}

func sampleTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testContext(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func recordTestEntry(t *testing.T, db *gorm.DB, tm tenant.Model, accountId uint32, accountName string, ip string, hwid string, success bool, failureReason string) Model {
	t.Helper()
	m, err := create(db)(tm, accountId, accountName, ip, hwid, success, failureReason)
	if err != nil {
		t.Fatalf("Failed to record test entry: %v", err)
	}
	return m
}

func TestProcessorRecord(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	p := NewProcessor(l, ctx, db)
	m, err := p.Record(42, "testuser", "10.0.0.1", "HWID123", true, "")
	if err != nil {
		t.Fatalf("Failed to record login: %v", err)
	}

	if m.AccountId() != 42 {
		t.Errorf("AccountId mismatch. Expected 42, got %v", m.AccountId())
	}
	if m.AccountName() != "testuser" {
		t.Errorf("AccountName mismatch. Expected testuser, got %v", m.AccountName())
	}
	if m.IPAddress() != "10.0.0.1" {
		t.Errorf("IPAddress mismatch. Expected 10.0.0.1, got %v", m.IPAddress())
	}
	if m.HWID() != "HWID123" {
		t.Errorf("HWID mismatch. Expected HWID123, got %v", m.HWID())
	}
	if m.Success() != true {
		t.Errorf("Success mismatch. Expected true, got %v", m.Success())
	}
}

func TestProcessorRecordFailedLogin(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	p := NewProcessor(l, ctx, db)
	m, err := p.Record(42, "testuser", "10.0.0.1", "HWID123", false, "invalid password")
	if err != nil {
		t.Fatalf("Failed to record login: %v", err)
	}

	if m.Success() != false {
		t.Errorf("Success mismatch. Expected false, got %v", m.Success())
	}
	if m.FailureReason() != "invalid password" {
		t.Errorf("FailureReason mismatch. Expected 'invalid password', got %v", m.FailureReason())
	}
}

func TestProcessorGetByAccountId(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	recordTestEntry(t, db, st, 42, "testuser", "10.0.0.1", "HWID1", true, "")
	recordTestEntry(t, db, st, 42, "testuser", "10.0.0.2", "HWID2", true, "")
	recordTestEntry(t, db, st, 99, "otheruser", "10.0.0.3", "HWID3", true, "")

	p := NewProcessor(l, ctx, db)
	entries, err := p.GetByAccountId(42)
	if err != nil {
		t.Fatalf("Failed to get by account id: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for account 42, got %d", len(entries))
	}
}

func TestProcessorGetByIP(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	recordTestEntry(t, db, st, 42, "testuser", "10.0.0.1", "HWID1", true, "")
	recordTestEntry(t, db, st, 43, "user2", "10.0.0.1", "HWID2", true, "")
	recordTestEntry(t, db, st, 44, "user3", "10.0.0.2", "HWID3", true, "")

	p := NewProcessor(l, ctx, db)
	entries, err := p.GetByIP("10.0.0.1")
	if err != nil {
		t.Fatalf("Failed to get by IP: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for IP 10.0.0.1, got %d", len(entries))
	}
}

func TestProcessorGetByHWID(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	recordTestEntry(t, db, st, 42, "testuser", "10.0.0.1", "ABC123", true, "")
	recordTestEntry(t, db, st, 43, "user2", "10.0.0.2", "ABC123", true, "")
	recordTestEntry(t, db, st, 44, "user3", "10.0.0.3", "XYZ789", true, "")

	p := NewProcessor(l, ctx, db)
	entries, err := p.GetByHWID("ABC123")
	if err != nil {
		t.Fatalf("Failed to get by HWID: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for HWID ABC123, got %d", len(entries))
	}
}

func TestProcessorGetByTenant(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	recordTestEntry(t, db, st, 42, "testuser", "10.0.0.1", "HWID1", true, "")
	recordTestEntry(t, db, st, 43, "user2", "10.0.0.2", "HWID2", true, "")
	recordTestEntry(t, db, st, 44, "user3", "10.0.0.3", "HWID3", false, "banned")

	p := NewProcessor(l, ctx, db)
	entries, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get by tenant: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
}

func TestProcessorGetByTenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st1 := sampleTenant()
	st2 := sampleTenant()

	recordTestEntry(t, db, st1, 42, "testuser", "10.0.0.1", "HWID1", true, "")
	recordTestEntry(t, db, st1, 43, "user2", "10.0.0.2", "HWID2", true, "")
	recordTestEntry(t, db, st2, 44, "user3", "10.0.0.3", "HWID3", true, "")

	p := NewProcessor(l, testContext(st1), db)
	entries, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Tenant isolation failed. Expected 2 entries for tenant 1, got %d", len(entries))
	}
}

func TestProcessorPurgeOlderThan(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	// Create an old entry by inserting directly with a past CreatedAt
	oldEntry := &Entity{
		TenantId:    st.Id(),
		AccountId:   42,
		AccountName: "testuser",
		IPAddress:   "10.0.0.1",
		HWID:        "HWID1",
		Success:     true,
		CreatedAt:   time.Now().AddDate(0, 0, -100),
	}
	if err := db.Create(oldEntry).Error; err != nil {
		t.Fatalf("Failed to create old entry: %v", err)
	}

	// Create a recent entry
	recordTestEntry(t, db, st, 43, "user2", "10.0.0.2", "HWID2", true, "")

	p := NewProcessor(l, ctx, db)
	err := p.PurgeOlderThan(90)
	if err != nil {
		t.Fatalf("Failed to purge: %v", err)
	}

	entries, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get entries after purge: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after purge, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].AccountId() != 43 {
		t.Errorf("Wrong entry survived purge. Expected account 43, got %v", entries[0].AccountId())
	}
}

func TestProcessorPurgeIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st1 := sampleTenant()
	st2 := sampleTenant()

	// Create old entries for both tenants
	oldEntry1 := &Entity{
		TenantId:    st1.Id(),
		AccountId:   42,
		AccountName: "user1",
		IPAddress:   "10.0.0.1",
		HWID:        "HWID1",
		Success:     true,
		CreatedAt:   time.Now().AddDate(0, 0, -100),
	}
	oldEntry2 := &Entity{
		TenantId:    st2.Id(),
		AccountId:   43,
		AccountName: "user2",
		IPAddress:   "10.0.0.2",
		HWID:        "HWID2",
		Success:     true,
		CreatedAt:   time.Now().AddDate(0, 0, -100),
	}
	if err := db.Create(oldEntry1).Error; err != nil {
		t.Fatalf("Failed to create old entry 1: %v", err)
	}
	if err := db.Create(oldEntry2).Error; err != nil {
		t.Fatalf("Failed to create old entry 2: %v", err)
	}

	// Purge only tenant 1
	p := NewProcessor(l, testContext(st1), db)
	err := p.PurgeOlderThan(90)
	if err != nil {
		t.Fatalf("Failed to purge: %v", err)
	}

	// Tenant 2 should still have its old entry
	p2 := NewProcessor(l, testContext(st2), db)
	entries, err := p2.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get tenant 2 entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Purge isolation failed. Expected 1 entry for tenant 2, got %d", len(entries))
	}
}
