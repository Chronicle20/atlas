package document

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type AllGetter[I uint32, M Identifier[I]] interface {
	All(ctx context.Context) model.Provider[[]M]
}

type ByIdGetter[I uint32, M Identifier[I]] interface {
	ById(ctx context.Context) func(id I) model.Provider[M]
}

type Adder[I uint32, M Identifier[I]] interface {
	Add(ctx context.Context) func(m M) model.Provider[M]
}

type Clearer[I uint32, M Identifier[I]] interface {
	Clear(ctx context.Context) error
}

type Storer[I uint32, M Identifier[I]] interface {
	AllGetter[I, M]
	ByIdGetter[I, M]
	Adder[I, M]
}

type RegStorage[I uint32, M Identifier[I]] struct {
	l logrus.FieldLogger
	r *Registry[I, M]
}

func NewRegStorage[I uint32, M Identifier[I]](l logrus.FieldLogger, r *Registry[I, M]) *RegStorage[I, M] {
	return &RegStorage[I, M]{
		l: l,
		r: r,
	}
}

func (s *RegStorage[I, M]) All(ctx context.Context) model.Provider[[]M] {
	t := tenant.MustFromContext(ctx)
	rms, err := s.r.GetAll(t)
	if err != nil {
		return model.ErrorProvider[[]M](err)
	}
	return model.FixedProvider(rms)
}

func (s *RegStorage[I, M]) ById(ctx context.Context) func(id I) model.Provider[M] {
	return func(id I) model.Provider[M] {
		t := tenant.MustFromContext(ctx)
		rm, err := s.r.Get(t, id)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.FixedProvider(rm)
	}
}

func (s *RegStorage[I, M]) Add(ctx context.Context) func(m M) model.Provider[M] {
	return func(m M) model.Provider[M] {
		t := tenant.MustFromContext(ctx)
		rm, err := s.r.Add(t, m)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.FixedProvider(rm)
	}
}

func (s *RegStorage[I, M]) Clear(ctx context.Context) error {
	t := tenant.MustFromContext(ctx)
	return s.r.Clear(t)
}
