package minioreconcile

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// fakeStore is an in-memory Store. prefixes[bucket][tenantID] = PrefixInfo.
type fakeStore struct {
	buckets    []string
	prefixes   map[string]map[string]PrefixInfo
	removed    []string         // "bucket:tenants/<id>/"
	removeErrs map[string]error // tenantID -> error RemovePrefix should return for that id
}

func (f *fakeStore) Buckets() []string { return f.buckets }
func (f *fakeStore) ListTenantIDs(_ context.Context, bucket string) ([]string, error) {
	ids := make([]string, 0)
	for id := range f.prefixes[bucket] {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic order for tests that assert partial progress
	return ids, nil
}
func (f *fakeStore) PrefixInfo(_ context.Context, bucket, prefix string) (PrefixInfo, error) {
	id := prefix[len("tenants/") : len(prefix)-1] // strip "tenants/" and trailing "/"
	return f.prefixes[bucket][id], nil
}
func (f *fakeStore) RemovePrefix(_ context.Context, bucket, prefix string) error {
	id := prefix[len("tenants/") : len(prefix)-1] // strip "tenants/" and trailing "/"
	if err, ok := f.removeErrs[id]; ok {
		return err
	}
	f.removed = append(f.removed, bucket+":"+prefix)
	return nil
}

const canonicalUUID = "00000000-0000-0000-0000-000000000000"

func now() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }

// old returns a timestamp h hours before now().
func old(h int) time.Time { return now().Add(-time.Duration(h) * time.Hour) }

func storeWith(rows map[string]PrefixInfo) *fakeStore {
	return &fakeStore{
		buckets:  []string{"atlas-wz"},
		prefixes: map[string]map[string]PrefixInfo{"atlas-wz": rows},
	}
}

func TestReconcile_EmptyKeepListRefused(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{"aaaa": {Count: 1, Bytes: 10, Newest: old(100)}})
	_, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: nil, MinAgeHours: 48}, now())
	if err != ErrEmptyKeepList {
		t.Fatalf("want ErrEmptyKeepList, got %v", err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("nothing must be removed on refusal, got %v", s.removed)
	}
}

func TestReconcile_KeepListPreserved(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{
		"keep-me": {Count: 1, Bytes: 10, Newest: old(100)},
		"orphan":  {Count: 2, Bytes: 20, Newest: old(100)},
	})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep-me"}, MinAgeHours: 48, DryRun: false}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 1 || s.removed[0] != "atlas-wz:tenants/orphan/" {
		t.Fatalf("only orphan should be removed, got %v", s.removed)
	}
	if rep.TotalPrefixes != 1 || rep.TotalBytes != 20 {
		t.Fatalf("report totals wrong: %+v", rep)
	}
}

func TestReconcile_CanonicalExcluded(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{canonicalUUID: {Count: 1, Bytes: 10, Newest: old(100)}})
	_, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"someone"}, MinAgeHours: 48}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("canonical sentinel must never be removed, got %v", s.removed)
	}
}

func TestReconcile_AgeGuardBoundary(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{
		"too-new": {Count: 1, Bytes: 10, Newest: old(47)}, // 47h < 48h → kept
		"exactly": {Count: 1, Bytes: 10, Newest: old(48)}, // 48h == 48h → eligible (pins the >= boundary)
		"old":     {Count: 1, Bytes: 10, Newest: old(49)}, // 49h ≥ 48h → eligible
	})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep"}, MinAgeHours: 48, DryRun: false}, now())
	if err != nil {
		t.Fatal(err)
	}
	removedSet := map[string]bool{}
	for _, r := range s.removed {
		removedSet[r] = true
	}
	if len(s.removed) != 2 || !removedSet["atlas-wz:tenants/exactly/"] || !removedSet["atlas-wz:tenants/old/"] {
		t.Fatalf("both the ==48h and >48h prefixes should be removed, got %v", s.removed)
	}
	// too-new is reported as kept-too-new, not deleted
	var keptTooNew int
	for _, row := range rep.Rows {
		if row.Action == "kept-too-new" {
			keptTooNew++
		}
	}
	if keptTooNew != 1 {
		t.Fatalf("want 1 kept-too-new row, got %d (%+v)", keptTooNew, rep.Rows)
	}
}

// TestReconcile_PartialReportOnRemoveError pins Fix 1: when RemovePrefix fails
// mid-sweep, Reconcile must return the Report accumulated so far (not an empty
// one), because earlier prefixes in the same sweep may already have been
// deleted — a real side effect the caller needs visibility into.
func TestReconcile_PartialReportOnRemoveError(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{
		"a-first":  {Count: 1, Bytes: 10, Newest: old(100)},
		"b-second": {Count: 2, Bytes: 20, Newest: old(100)},
	})
	wantErr := errors.New("minio: remove failed")
	s.removeErrs = map[string]error{"b-second": wantErr}

	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep"}, MinAgeHours: 48, DryRun: false}, now())

	if err == nil {
		t.Fatal("want non-nil error when RemovePrefix fails")
	}
	if len(s.removed) != 1 || s.removed[0] != "atlas-wz:tenants/a-first/" {
		t.Fatalf("a-first should already be removed before b-second errored, got %v", s.removed)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].TenantID != "a-first" || rep.Rows[0].Action != "deleted" {
		t.Fatalf("returned Report must retain the already-deleted a-first row, got %+v", rep.Rows)
	}
	if rep.TotalPrefixes != 1 || rep.TotalBytes != 10 {
		t.Fatalf("returned Report totals must reflect the partial progress, got %+v", rep)
	}
}

func TestReconcile_DryRunDeletesNothing(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{"orphan": {Count: 1, Bytes: 10, Newest: old(100)}})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep"}, MinAgeHours: 48, DryRun: true}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("dryRun must not remove, got %v", s.removed)
	}
	if rep.TotalPrefixes != 1 || rep.Rows[0].Action != "would-delete" {
		t.Fatalf("dryRun should report would-delete: %+v", rep)
	}
}
