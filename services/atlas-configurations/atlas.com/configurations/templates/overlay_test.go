package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// envContext installs a registry (task-232 R13-2: env.CurrentRegistry) that
// knows "main", "pr-123" and "pr-999" - all baselined to "main" - and
// returns a context carrying caller as the operation's environment. The
// registry is process-wide, so the previous one is restored on cleanup to
// avoid leaking state into other tests in this package.
func envContext(t *testing.T, caller string) context.Context {
	t.Helper()
	reg := env.NewMapRegistry(env.Id(caller), nil)
	for _, e := range []string{"main", "pr-123", "pr-999"} {
		reg.Apply(env.Record{Name: env.Id(e), Baseline: env.Id("main"), Phase: env.PhaseActive})
	}
	prev := env.CurrentRegistry()
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(prev) })
	return env.WithContext(context.Background(), env.Id(caller))
}

// seedTemplate inserts one template row directly at the Entity level (via
// the SQLite-compatible testEntity already established by
// processor_test.go's setupTestDB) and returns it.
func seedTemplate(t *testing.T, db *gorm.DB, environment string, region string, majorVersion uint16, minorVersion uint16) testEntity {
	t.Helper()
	// Make() sources Region/MajorVersion/MinorVersion from the JSON Data
	// blob, not from the Entity columns - mirror that here so a seeded
	// row's RestModel round-trips with the version key its Entity columns
	// (and the overlay's WHERE clause) were seeded with.
	data, err := json.Marshal(map[string]any{
		"region":       region,
		"majorVersion": majorVersion,
		"minorVersion": minorVersion,
	})
	if err != nil {
		t.Fatalf("failed to marshal seed data: %v", err)
	}
	e := testEntity{
		Id:           uuid.New(),
		Region:       region,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		Data:         data,
		Environment:  environment,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("failed to seed template for environment %q: %v", environment, err)
	}
	return e
}

func TestTemplatesFallBackToTheBaselineRow(t *testing.T) {
	db := setupTestDB(t)
	// Only main has a v83.1 template.
	seedTemplate(t, db, "main", "GMS", 83, 1)

	got, err := NewProcessor(testLogger(), envContext(t, "pr-123"), db).
		GetByRegionAndVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("GetByRegionAndVersion: %v", err)
	}
	if got.Environment != "main" {
		t.Fatalf("pr-123 got environment %q, want the baseline's row", got.Environment)
	}
}

func TestTemplatesPreferTheOwnEnvironmentRow(t *testing.T) {
	db := setupTestDB(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := NewProcessor(testLogger(), envContext(t, "pr-123"), db).
		GetByRegionAndVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("GetByRegionAndVersion: %v", err)
	}
	if got.Environment != "pr-123" {
		t.Fatalf("got environment %q, want pr-123's own row to win", got.Environment)
	}
}

func TestTemplateCollectionIsAnOverlayNotAUnion(t *testing.T) {
	// The case an ORDER BY cannot express. main ships two versions; pr-123
	// overrides one of them. The collection read must return pr-123's 83.1
	// and main's 95.1 - two rows, not three.
	db := setupTestDB(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "main", "GMS", 95, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := NewProcessor(testLogger(), envContext(t, "pr-123"), db).
		AllProvider(model.Page{Number: 1, Size: 50})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("got %d rows, want 2 (overlay, not union): %+v", len(got.Items), got.Items)
	}
	byVersion := map[string]string{}
	for _, e := range got.Items {
		byVersion[fmt.Sprintf("%s%d.%d", e.Region, e.MajorVersion, e.MinorVersion)] = e.Environment
	}
	if byVersion["GMS83.1"] != "pr-123" {
		t.Fatalf("GMS83.1 came from %q, want pr-123's overriding row", byVersion["GMS83.1"])
	}
	if byVersion["GMS95.1"] != "main" {
		t.Fatalf("GMS95.1 came from %q, want the inherited baseline row", byVersion["GMS95.1"])
	}
}

func TestTemplateCollectionOnMainIsUnchanged(t *testing.T) {
	// NG6/NFR-7: the baseline's own collection read must be byte-identical
	// to today's - it inherits from nothing and must not see other
	// environments.
	db := setupTestDB(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "main", "GMS", 95, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := NewProcessor(testLogger(), envContext(t, "main"), db).
		AllProvider(model.Page{Number: 1, Size: 50})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("main saw %d rows, want its own 2", len(got.Items))
	}
	for _, e := range got.Items {
		if e.Environment != "main" {
			t.Fatalf("main's collection read returned a %q row", e.Environment)
		}
	}
}

func TestTemplateByIdRejectsAnotherEnvironmentsRow(t *testing.T) {
	// A UUID is unique, so there is nothing to fall back to. pr-123 may
	// read its own rows and its baseline's (templates are a shared
	// read-only source), and nothing else.
	db := setupTestDB(t)
	mine := seedTemplate(t, db, "pr-123", "GMS", 83, 1)
	inherited := seedTemplate(t, db, "main", "GMS", 95, 1)
	foreign := seedTemplate(t, db, "pr-999", "GMS", 83, 1)

	p := NewProcessor(testLogger(), envContext(t, "pr-123"), db)
	if _, err := p.GetById(mine.Id); err != nil {
		t.Fatalf("own row: %v", err)
	}
	if _, err := p.GetById(inherited.Id); err != nil {
		t.Fatalf("baseline row: %v", err)
	}
	if _, err := p.GetById(foreign.Id); err == nil {
		t.Fatal("read another environment's template by id; want not-found")
	}
}

// TestOverlayCollectionAntiJoinTargetsTheUnaliasedTemplatesTable pins the
// brief's explicit warning: if database.PagedQuery ever introduces a table
// alias, the anti-join's correlated subquery (which hard-codes "templates")
// silently stops correlating and the overlay degenerates into a union. This
// asserts against the ACTUAL generated SQL rather than assuming GORM's
// behaviour.
func TestOverlayCollectionAntiJoinTargetsTheUnaliasedTemplatesTable(t *testing.T) {
	db := setupTestDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var rows []Entity
		return OverlayCollection(tx.Model(&Entity{}), env.Id("pr-123"), env.Id("main")).Find(&rows)
	})

	if !strings.Contains(sql, `FROM "templates"`) && !strings.Contains(sql, "FROM templates") {
		t.Fatalf("expected the outer query to select from the unaliased templates table, got: %s", sql)
	}
	if strings.Contains(sql, "AS `templates`") || strings.Contains(sql, `AS "templates"`) {
		t.Fatalf("templates table is aliased; the anti-join's correlated subquery hard-codes \"templates\" and would silently stop correlating: %s", sql)
	}
}
