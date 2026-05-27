package seeder

import (
	"regexp"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

type Subdomain[J any, M any] interface {
	Name() string
	Path() string
	Type() string
	EntityIDPattern() *regexp.Regexp
	DeleteAllForTenant(db *gorm.DB) (int64, error)
	Decode(payload []byte) (J, error)
	Build(t tenant.Model, entityID string, j J) ([]M, error)
	BulkCreate(db *gorm.DB, models []M) error
	Count(db *gorm.DB) (count int64, mostRecentUpdate *time.Time, err error)
}

// SubdomainAuxiliary is an OPTIONAL extension a Subdomain implementation
// can satisfy when its Build creates rows in additional tables as a side
// effect of seeding (e.g. npc-shops' Build also writes the commodities
// table — same delete/build/create cycle, different table).
//
// Implementing this lets the status endpoint surface counts for those
// auxiliary tables under their own keys in the response's `subdomains`
// map. Without it, the auxiliary table's contents are invisible to the
// UI even though they're maintained by the same seed run.
//
// The returned map is merged into the status response after the primary
// Count(). Keys must NOT collide with any subdomain Name() in the same
// Group; the merge skips duplicates and logs a warning.
type SubdomainAuxiliary interface {
	AuxiliaryCounts(db *gorm.DB) (map[string]SubdomainStatus, error)
}

type SubdomainAny interface {
	Name() string
	Path() string
	Type() string
	EntityIDPattern() *regexp.Regexp
	DeleteAllForTenant(db *gorm.DB) (int64, error)
	LoadAndBuild(t tenant.Model, entityID string, payload []byte) (any, error)
	BulkCreate(db *gorm.DB, rows any) error
	Count(db *gorm.DB) (int64, *time.Time, error)
	// AuxiliaryCounts returns extra row counts for tables the subdomain
	// maintains as side effects. Returns (nil, nil) when the inner
	// Subdomain does not implement SubdomainAuxiliary.
	AuxiliaryCounts(db *gorm.DB) (map[string]SubdomainStatus, error)
}

type adapter[J any, M any] struct {
	inner Subdomain[J, M]
}

func AdaptSubdomain[J any, M any](s Subdomain[J, M]) SubdomainAny {
	return &adapter[J, M]{inner: s}
}

func (a *adapter[J, M]) Name() string                    { return a.inner.Name() }
func (a *adapter[J, M]) Path() string                    { return a.inner.Path() }
func (a *adapter[J, M]) Type() string                    { return a.inner.Type() }
func (a *adapter[J, M]) EntityIDPattern() *regexp.Regexp { return a.inner.EntityIDPattern() }

func (a *adapter[J, M]) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return a.inner.DeleteAllForTenant(db)
}

func (a *adapter[J, M]) LoadAndBuild(t tenant.Model, entityID string, payload []byte) (any, error) {
	j, err := a.inner.Decode(payload)
	if err != nil {
		return nil, err
	}
	rows, err := a.inner.Build(t, entityID, j)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (a *adapter[J, M]) BulkCreate(db *gorm.DB, rows any) error {
	typed, ok := rows.([]M)
	if !ok {
		return errAdapterTypeMismatch
	}
	return a.inner.BulkCreate(db, typed)
}

func (a *adapter[J, M]) Count(db *gorm.DB) (int64, *time.Time, error) {
	return a.inner.Count(db)
}

func (a *adapter[J, M]) AuxiliaryCounts(db *gorm.DB) (map[string]SubdomainStatus, error) {
	if aux, ok := any(a.inner).(SubdomainAuxiliary); ok {
		return aux.AuxiliaryCounts(db)
	}
	return nil, nil
}

var errAdapterTypeMismatch = errAdapterMismatch{}

type errAdapterMismatch struct{}

func (errAdapterMismatch) Error() string {
	return "atlas-seeder: BulkCreate received rows of unexpected element type"
}
