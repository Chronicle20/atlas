package document

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Storage[I uint32, M Identifier[I]] struct {
	l      logrus.FieldLogger
	regSto *RegStorage[I, M]
	dbSto  *DbStorage[I, M]
}

func NewStorage[I uint32, M Identifier[I]](l logrus.FieldLogger, db *gorm.DB, r *Registry[I, M], docType string) *Storage[I, M] {
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
		nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
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
	if err == nil {
		return model.FixedProvider(ms)
	}
	nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
	if err != nil {
		return model.ErrorProvider[[]M](err)
	}
	nctx := tenant.WithContext(ctx, nt)
	ms, err = s.dbSto.All(nctx)()
	if err == nil {
		return model.FixedProvider(ms)
	}
	return model.ErrorProvider[[]M](err)
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
