package document

import (
	"atlas-data/canonical"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Storage[I string, M Identifier[I]] struct {
	l      logrus.FieldLogger
	regSto *RegStorage[I, M]
	dbSto  *DbStorage[I, M]
}

func NewStorage[I string, M Identifier[I]](l logrus.FieldLogger, db *gorm.DB, r *Registry[I, M], docType string) *Storage[I, M] {
	return &Storage[I, M]{
		l:      l,
		regSto: NewRegStorage(l, r),
		dbSto:  NewDbStorage[I, M](l, db, docType),
	}
}

func (s *Storage[I, M]) ByIdProvider(ctx context.Context) func(id I) model.Provider[M] {
	t := tenant.MustFromContext(ctx)
	return func(id I) model.Provider[M] {
		var m M
		var err error
		m, err = s.regSto.ById(ctx)(id)()
		if err == nil {
			return model.FixedProvider(m)
		}
		m, err = s.dbSto.ById(ctx)(id)()
		if err == nil {
			_, err = s.regSto.Add(ctx)(m)()
			if err != nil {
				return model.ErrorProvider[M](err)
			}
			return model.FixedProvider(m)
		}
		nt, err := tenant.Create(canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), t.Region(), t.MajorVersion(), t.MinorVersion())
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		nctx := tenant.WithContext(ctx, nt)
		m, err = s.regSto.ById(nctx)(id)()
		if err == nil {
			return model.FixedProvider(m)
		}
		m, err = s.dbSto.ById(nctx)(id)()
		if err == nil {
			_, err = s.regSto.Add(nctx)(m)()
			if err != nil {
				return model.ErrorProvider[M](err)
			}
			return model.FixedProvider(m)
		}
		return model.ErrorProvider[M](err)
	}
}

func (s *Storage[I, M]) GetById(ctx context.Context) func(id I) (M, error) {
	return func(id I) (M, error) {
		return s.ByIdProvider(ctx)(id)()
	}
}

func (s *Storage[I, M]) AllProvider(ctx context.Context) model.Provider[[]M] {
	t := tenant.MustFromContext(ctx)
	var ms []M
	var err error
	ms, err = s.dbSto.All(ctx)()
	if err == nil && len(ms) > 0 {
		return model.FixedProvider(ms)
	}
	// A tenant with no rows of this document type falls back to the
	// version-scoped canonical dataset (canonical.TenantId), mirroring
	// ByIdProvider. Without this, a tenant that was never directly seeded
	// (e.g. a version provisioned after canonical ingestion) gets an empty
	// batch result while per-id lookups silently succeed via the same
	// canonical fallback — an asymmetry that surfaced as preset skill
	// validation rejecting every skill for such tenants.
	nt, cerr := tenant.Create(canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), t.Region(), t.MajorVersion(), t.MinorVersion())
	if cerr != nil {
		return model.ErrorProvider[[]M](cerr)
	}
	nctx := tenant.WithContext(ctx, nt)
	cms, cerr := s.dbSto.All(nctx)()
	if cerr == nil && len(cms) > 0 {
		return model.FixedProvider(cms)
	}
	// No canonical rows either: prefer a real error from the original lookup,
	// otherwise return the (empty) original result rather than masking it.
	if err != nil {
		return model.ErrorProvider[[]M](err)
	}
	return model.FixedProvider(ms)
}

func (s *Storage[I, M]) GetAll(ctx context.Context) ([]M, error) {
	return s.AllProvider(ctx)()
}

func (s *Storage[I, M]) Add(ctx context.Context) func(m M) model.Provider[M] {
	return func(m M) model.Provider[M] {
		var err error
		_, err = s.dbSto.Add(ctx)(m)()
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		_, err = s.regSto.Add(ctx)(m)()
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.FixedProvider(m)
	}
}

func (s *Storage[I, M]) Logger() logrus.FieldLogger {
	return s.l
}
