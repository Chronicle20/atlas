package seeder

import (
	"context"
	"encoding/json"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

type widgetAttrs struct {
	Name string `json:"name"`
}
type widgetRow struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

type widgetSubdomain struct {
	deleted int64
	created atomic.Int64
}

func (w *widgetSubdomain) Name() string                    { return "widgets" }
func (w *widgetSubdomain) Path() string                    { return "widgets" }
func (w *widgetSubdomain) Type() string                    { return "widget" }
func (w *widgetSubdomain) EntityIDPattern() *regexp.Regexp { return regexp.MustCompile(`^widget-(\d+)\.json$`) }
func (w *widgetSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) {
	return w.deleted, nil
}
func (w *widgetSubdomain) Decode(b []byte) (widgetAttrs, error) {
	var a widgetAttrs
	return a, json.Unmarshal(b, &a)
}
func (w *widgetSubdomain) Build(_ tenant.Model, id string, a widgetAttrs) ([]widgetRow, error) {
	n, _ := uintFromString(id)
	return []widgetRow{{ID: n, Name: a.Name}}, nil
}
func (w *widgetSubdomain) BulkCreate(_ *gorm.DB, rows []widgetRow) error {
	w.created.Add(int64(len(rows)))
	return nil
}
func (w *widgetSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) {
	now := time.Now().UTC()
	return w.created.Load(), &now, nil
}

func uintFromString(s string) (uint64, error) {
	var n uint64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, nil
		}
		n = n*10 + uint64(r-'0')
	}
	return n, nil
}

func TestSeed_SuccessfulRunPersistsStateAndCountsCreated(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:      "widgets-group",
		URLPrefix: "/widgets",
		Subdomains: []SubdomainAny{
			AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{}),
		},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("revision = %q, want test-rev-abc123", res.CatalogRevision)
	}
	if res.Subdomains["widgets"].Created != 2 {
		t.Fatalf("created = %d, want 2 (widget-1.json + widget-2.json)", res.Subdomains["widgets"].Created)
	}
	tm2 := tenant.MustFromContext(ctx)
	row, err := ReadSeedState(db, tm2.Id(), "widgets-group")
	if err != nil || row == nil {
		t.Fatalf("expected seed_state row, got err=%v row=%v", err, row)
	}
	if row.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("row.CatalogRevision = %q", row.CatalogRevision)
	}
}

type failingSubdomain struct{}

func (f *failingSubdomain) Name() string                    { return "broken" }
func (f *failingSubdomain) Path() string                    { return "widgets" }
func (f *failingSubdomain) Type() string                    { return "widget" }
func (f *failingSubdomain) EntityIDPattern() *regexp.Regexp { return regexp.MustCompile(`^widget-(\d+)\.json$`) }
func (f *failingSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) { return 0, nil }
func (f *failingSubdomain) Decode(_ []byte) (widgetAttrs, error) {
	return widgetAttrs{}, errBad
}
func (f *failingSubdomain) Build(_ tenant.Model, _ string, _ widgetAttrs) ([]widgetRow, error) {
	return nil, nil
}
func (f *failingSubdomain) BulkCreate(_ *gorm.DB, _ []widgetRow) error  { return nil }
func (f *failingSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) { return 0, nil, nil }

var errBad = errSimple("intentional decode failure")

type errSimple string

func (e errSimple) Error() string { return string(e) }

func TestSeed_PartialFailurePersistsAndContinues(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:      "mixed",
		URLPrefix: "/mixed",
		Subdomains: []SubdomainAny{
			AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{}),
			AdaptSubdomain[widgetAttrs, widgetRow](&failingSubdomain{}),
		},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.Subdomains["widgets"].Created != 2 {
		t.Fatalf("widgets created = %d, want 2", res.Subdomains["widgets"].Created)
	}
	if res.Subdomains["broken"].Failed != 2 {
		t.Fatalf("broken failed = %d, want 2 (decode failures)", res.Subdomains["broken"].Failed)
	}
	tm3 := tenant.MustFromContext(ctx)
	row, _ := ReadSeedState(db, tm3.Id(), "mixed")
	if row == nil {
		t.Fatalf("seed_state row missing on partial failure")
	}
}
