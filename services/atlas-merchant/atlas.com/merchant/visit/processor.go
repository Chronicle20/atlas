package visit

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Record(shopId uuid.UUID, name string) error
	ListPaged(shopId uuid.UUID, page model.Page) (model.Paged[Model], error)
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

// ListPaged backs the GET /merchants/{shopId}/visits list route (task-117).
func (p *ProcessorImpl) ListPaged(shopId uuid.UUID, page model.Page) (model.Paged[Model], error) {
	ep := getByShopIdPaged(shopId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(func(e Entity) (Model, error) { return Model{name: e.Name, count: e.Count}, nil })(ep)(model.ParallelMap())()
}
