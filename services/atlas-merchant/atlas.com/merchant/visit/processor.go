package visit

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Record(shopId uuid.UUID, name string) error
	List(shopId uuid.UUID) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Record(shopId uuid.UUID, name string) error {
	if name == "" {
		return nil
	}
	_, err := upsertVisit(p.t.Id(), shopId, name)(p.db.WithContext(p.ctx))()
	return err
}

func (p *ProcessorImpl) List(shopId uuid.UUID) ([]Model, error) {
	es, err := getByShopId(shopId)(p.db.WithContext(p.ctx))()
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(es))
	for _, e := range es {
		out = append(out, Model{name: e.Name, count: e.Count})
	}
	return out, nil
}
