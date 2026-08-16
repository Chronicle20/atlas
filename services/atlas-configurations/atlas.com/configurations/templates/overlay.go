package templates

import (
	"atlas-configurations/scope"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// Templates are the one control-plane table with a baseline-fallback
// scoping strategy (design deviation V3): a PR environment inherits every
// template it has not overridden, rather than seeing nothing until it
// clones the whole catalog. tenants and services stay strict (scope.Strict)
// because there is no meaningful "baseline row" for a tenant or a service -
// each one belongs to exactly one environment.
//
// overlayKey is the identity a template row is unique on. Fallback is
// defined ONLY in terms of this key: a baseline row is visible to e exactly
// when e has no row with the same key. Templates are keyed by a VERSION, not
// by an environment, and the PR bootstrap already treats them as a shared
// read-only source it clones from (design V3).
var overlayKey = []string{"region", "major_version", "minor_version"}

// OverlaySingle scopes a lookup of one version key. e's row wins; the
// baseline's fills in when e has none.
//
// The CASE ordering must be a clause.OrderByColumn (a raw column, not a
// clause.OrderBy{Expression: ...}): gorm's First() appends its own
// primary-key OrderBy after this one, and OrderBy.MergeClause only carries
// the PRIOR clause's Columns forward into the merged clause - a prior
// Expression-only OrderBy is silently dropped the moment a second Order()
// call composes with it. Encoding the CASE as a Columns entry keeps it
// through that merge; see templates/overlay_test.go's
// TestOverlaySingleCaseOrderSurvivesFirstsPrimaryKeyOrder for the SQL-level
// pin. e is an env.Id resolved from the trusted environment registry
// (task-232 R13-2), never client-supplied text, but the quote is still
// escaped defensively before being embedded as a raw SQL literal.
func OverlaySingle(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	escaped := strings.ReplaceAll(string(e), "'", "''")
	return db.Where("environment IN (?, ?)", string(e), string(baseline)).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: fmt.Sprintf("CASE WHEN environment = '%s' THEN 0 ELSE 1 END", escaped),
				Raw:  true,
			},
		})
}

// OverlayCollection scopes a Find/paged read: e's rows, plus the baseline
// rows whose version key e has no row for. This is an anti-join - an ORDER
// BY cannot express it, because a collection read returns every matching
// row rather than the first.
//
// NOT EXISTS rather than DISTINCT ON: it composes with database.PagedQuery's
// LIMIT/OFFSET, and it runs on both Postgres and the sqlite test harness.
func OverlayCollection(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	keyConditions := make([]string, len(overlayKey))
	for i, k := range overlayKey {
		keyConditions[i] = fmt.Sprintf("o.%s = templates.%s", k, k)
	}
	anti := fmt.Sprintf(`environment = ? OR (environment = ? AND NOT EXISTS (
	           SELECT 1 FROM templates o
	           WHERE o.environment = ?
	             AND %s))`, strings.Join(keyConditions, " AND "))
	return db.Where(anti, string(e), string(baseline), string(e))
}

// VisibleById scopes a UUID lookup. A UUID is unique, so there is nothing to
// fall back to - this is a visibility rule, not an overlay: e may read its
// own rows and its baseline's, and nothing else.
func VisibleById(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	return db.Where("environment IN (?, ?)", string(e), string(baseline))
}
