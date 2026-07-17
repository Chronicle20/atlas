package baseline

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestPendingRestoresSelectsOnlyRestoring verifies pendingRestores returns every
// StatusRestoring tenant (with its region/version) and none that are complete —
// the input Reconcile uses to decide which interrupted restores to re-run.
func TestPendingRestoresSelectsOnlyRestoring(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE tenant_baselines (
		tenant_id TEXT PRIMARY KEY, region TEXT, major_version INTEGER,
		minor_version INTEGER, baseline_sha256 TEXT, restored_at TEXT, status TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		tid, region, status string
		major, minor        int
	}{
		{"11111111-1111-1111-1111-111111111111", "GMS", StatusRestoring, 83, 1},
		{"22222222-2222-2222-2222-222222222222", "GMS", StatusComplete, 83, 1},
		{"33333333-3333-3333-3333-333333333333", "JMS", StatusRestoring, 185, 1},
	}
	for _, r := range rows {
		if err := db.Exec(
			`INSERT INTO tenant_baselines (tenant_id, region, major_version, minor_version, baseline_sha256, restored_at, status) VALUES (?, ?, ?, ?, 'sha', '', ?)`,
			r.tid, r.region, r.major, r.minor, r.status,
		).Error; err != nil {
			t.Fatal(err)
		}
	}

	pending, err := pendingRestores(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending restores, got %d: %+v", len(pending), pending)
	}
	got := map[string]restoreIntent{}
	for _, p := range pending {
		got[p.TenantID] = p
	}
	if _, ok := got["22222222-2222-2222-2222-222222222222"]; ok {
		t.Fatal("complete tenant must not be returned as pending")
	}
	jms, ok := got["33333333-3333-3333-3333-333333333333"]
	if !ok {
		t.Fatal("restoring JMS tenant missing from pending set")
	}
	if jms.Region != "JMS" || jms.MajorVersion != 185 || jms.MinorVersion != 1 {
		t.Fatalf("pending intent carries wrong version: %+v", jms)
	}
}

// TestReconcileSkipsWithoutMinio verifies Reconcile is a safe no-op when the
// MinIO client is unavailable (it can't restore without the dump source) and
// does not touch the DB — so it must not panic on a nil db either.
func TestReconcileSkipsWithoutMinio(t *testing.T) {
	Reconcile(context.Background(), logrus.New(), nil, nil)
}

// TestAdvisoryKeyStableAndDistinct pins that the per-tenant advisory-lock key is
// deterministic per tenant and differs between tenants, so the lock serializes
// the right tenant without colliding across tenants.
func TestAdvisoryKeyStableAndDistinct(t *testing.T) {
	a := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if advisoryKey(a) != advisoryKey(a) {
		t.Fatal("advisoryKey must be deterministic for the same tenant")
	}
	if advisoryKey(a) == advisoryKey(b) {
		t.Fatal("advisoryKey must differ between distinct tenants")
	}
}
