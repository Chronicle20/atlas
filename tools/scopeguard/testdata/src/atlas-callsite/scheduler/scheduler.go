// Package scheduler reproduces the atlas-marriages shape task-232's
// Task 3B/9B review flagged: a long-lived struct holding both a `ctx` and a
// `db` field, where one call site on `db` correctly chains `.WithContext(`
// and a sibling call site in the very same package does not. A per-service
// aggregate check ("this package has WithContext calls, therefore it's
// fine") cannot see the second call site; scopeguard must, because it works
// per call site.
package scheduler

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// gormDB is a minimal stand-in for *gorm.DB's fluent query builder. The
// scopeguard call-site rule is deliberately field-name-syntactic (matches
// on a field literally named `db`/`DB`), not type-checked against gorm — so
// this stub needs only the same method names and chaining shape, not the
// real type.
type gormDB struct{}

func (g *gormDB) WithContext(ctx context.Context) *gormDB     { return g }
func (g *gormDB) Model(v interface{}) *gormDB                 { return g }
func (g *gormDB) Where(q string, args ...interface{}) *gormDB { return g }
func (g *gormDB) Pluck(col string, dest interface{}) *gormDB  { return g }

type Scheduler struct {
	ctx context.Context
	db  *gormDB
}

// wrapped is the correct shape: the chain reaches .WithContext before any
// GORM verb. No diagnostic expected.
func (s *Scheduler) wrapped() {
	var tenantIds []string
	s.db.WithContext(s.ctx).Model(nil).Where("status = ?", "pending").Pluck("tenant_id", &tenantIds)
}

// unwrapped is the atlas-marriages defect shape: same struct, same package
// as wrapped above, but this call site never reaches .WithContext — it
// collapses to context.Background() and the fleet-wide tenant callback
// silently no-ops.
func (s *Scheduler) unwrapped() {
	var tenantIds []string
	s.db.Model(nil).Where("status = ?", "pending").Pluck("tenant_id", &tenantIds) // want "GORM call on a db field with no"
}

// optOut is the other mechanism: an explicit WithoutTenantFilter call site
// with no allowlist entry.
func (s *Scheduler) optOut() {
	noTenantCtx := database.WithoutTenantFilter(s.ctx) // want "call site opts out of tenant scoping"
	s.db.WithContext(noTenantCtx).Where("permanent = ?", false)
}
