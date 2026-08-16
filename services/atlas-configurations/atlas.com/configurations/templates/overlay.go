package templates

import (
	"atlas-configurations/scope"

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
func OverlaySingle(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	return db.Where("environment IN (?, ?)", string(e), string(baseline)).
		Order(clause.Expr{
			SQL:  "CASE WHEN environment = ? THEN 0 ELSE 1 END",
			Vars: []interface{}{string(e)},
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
	anti := `environment = ? OR (environment = ? AND NOT EXISTS (
	           SELECT 1 FROM templates o
	           WHERE o.environment = ?
	             AND o.region = templates.region
	             AND o.major_version = templates.major_version
	             AND o.minor_version = templates.minor_version))`
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
