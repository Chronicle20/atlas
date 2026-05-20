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

type SubdomainAny interface {
	Name() string
	Path() string
	Type() string
	EntityIDPattern() *regexp.Regexp
	DeleteAllForTenant(db *gorm.DB) (int64, error)
	LoadAndBuild(t tenant.Model, entityID string, payload []byte) (any, error)
	BulkCreate(db *gorm.DB, rows any) error
	Count(db *gorm.DB) (int64, *time.Time, error)
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

var errAdapterTypeMismatch = errAdapterMismatch{}

type errAdapterMismatch struct{}

func (errAdapterMismatch) Error() string {
	return "atlas-seeder: BulkCreate received rows of unexpected element type"
}
