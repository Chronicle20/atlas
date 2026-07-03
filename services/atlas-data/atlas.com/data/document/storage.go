package document

import (
	"context"

	"atlas-data/canonical"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

// AllPagedProvider pages this document type for the context tenant. A tenant
// page with Total 0 falls back to the version-scoped canonical dataset
// (canonical.TenantId), mirroring AllProvider's fallback so the paged
// variant does not reintroduce the batch-GetAll-skips-fallback asymmetry
// (see AllProvider's comment / PR #759): a version provisioned after
// canonical ingestion has no per-tenant rows, so per-id lookups would
// silently succeed via canonical while a naive paged batch read stayed
// empty.
func (s *Storage[I, M]) AllPagedProvider(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]] {
	t := tenant.MustFromContext(ctx)
	return func(page model.Page) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			p, err := s.dbSto.AllPaged(ctx)(page)()
			if err == nil && p.Total > 0 {
				return p, nil
			}
			nt, cerr := tenant.Create(canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), t.Region(), t.MajorVersion(), t.MinorVersion())
			if cerr != nil {
				return model.Paged[M]{}, cerr
			}
			nctx := tenant.WithContext(ctx, nt)
			cp, cerr := s.dbSto.AllPaged(nctx)(page)()
			if cerr == nil && cp.Total > 0 {
				return cp, nil
			}
			// No canonical rows either: prefer a real error from the original
			// lookup, otherwise return the (empty) original result rather than
			// masking it.
			if err != nil {
				return model.Paged[M]{}, err
			}
			return p, nil
		}
	}
}

// DrainAllProvider accumulates every document of this type (tenant scope
// with canonical fallback) by paging internally. For in-process callers
// that genuinely need the full set (e.g. search-index builds).
func (s *Storage[I, M]) DrainAllProvider(ctx context.Context) model.Provider[[]M] {
	return func() ([]M, error) {
		const drainPageSize = 1000
		var out []M
		for number := 1; ; number++ {
			p, err := s.AllPagedProvider(ctx)(model.Page{Number: number, Size: drainPageSize})()
			if err != nil {
				return nil, err
			}
			out = append(out, p.Items...)
			if len(p.Items) == 0 || len(out) >= p.Total {
				return out, nil
			}
		}
	}
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
