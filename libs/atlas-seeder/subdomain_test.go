package seeder

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeAttrs struct {
	Name string `json:"name"`
}

type fakeRow struct {
	ID   uint64
	Name string
}

type fakeSubdomain struct {
	name       string
	path       string
	typ        string
	pattern    *regexp.Regexp
	decoded    fakeAttrs
	builtRows  []fakeRow
	deleted    int64
	count      int64
	updatedAt  *time.Time
	bulkCalled bool
}

func (f *fakeSubdomain) Name() string                    { return f.name }
func (f *fakeSubdomain) Path() string                    { return f.path }
func (f *fakeSubdomain) Type() string                    { return f.typ }
func (f *fakeSubdomain) EntityIDPattern() *regexp.Regexp { return f.pattern }
func (f *fakeSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) {
	return f.deleted, nil
}
func (f *fakeSubdomain) Decode(b []byte) (fakeAttrs, error) {
	var a fakeAttrs
	err := json.Unmarshal(b, &a)
	f.decoded = a
	return a, err
}
func (f *fakeSubdomain) Build(_ tenant.Model, _ string, a fakeAttrs) ([]fakeRow, error) {
	r := fakeRow{ID: 1, Name: a.Name}
	f.builtRows = append(f.builtRows, r)
	return []fakeRow{r}, nil
}
func (f *fakeSubdomain) BulkCreate(_ *gorm.DB, _ []fakeRow) error {
	f.bulkCalled = true
	return nil
}
func (f *fakeSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) {
	return f.count, f.updatedAt, nil
}

func TestAdaptSubdomain_PreservesNameAndPath(t *testing.T) {
	s := &fakeSubdomain{name: "widgets", path: "widgets", typ: "widget"}
	a := AdaptSubdomain[fakeAttrs, fakeRow](s)
	if a.Name() != "widgets" || a.Path() != "widgets" || a.Type() != "widget" {
		t.Fatalf("adapter dropped metadata: %s/%s/%s", a.Name(), a.Path(), a.Type())
	}
}

func TestAdaptSubdomain_DecodeBuildBulkCreatePropagate(t *testing.T) {
	s := &fakeSubdomain{
		name:    "widgets",
		path:    "widgets",
		typ:     "widget",
		pattern: regexp.MustCompile(`^widget-(\d+)\.json$`),
	}
	a := AdaptSubdomain[fakeAttrs, fakeRow](s)
	tm, err := tenant.Create(uuid.New(), "gms", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	rows, err := a.LoadAndBuild(tm, "42", []byte(`{"name":"hello"}`))
	if err != nil {
		t.Fatalf("LoadAndBuild: %v", err)
	}
	rowsTyped, ok := rows.([]fakeRow)
	if !ok {
		t.Fatalf("rows not []fakeRow: %T", rows)
	}
	if len(rowsTyped) != 1 {
		t.Fatalf("rows = %d, want 1", len(rowsTyped))
	}
	if err := a.BulkCreate(nil, rows); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}
	if !s.bulkCalled {
		t.Fatalf("inner BulkCreate not called")
	}
	if s.decoded.Name != "hello" {
		t.Fatalf("decoded name = %q", s.decoded.Name)
	}
}
