package blacklist

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Add(shopId uuid.UUID, name string) error
	Remove(shopId uuid.UUID, name string) error
	NamesPaged(shopId uuid.UUID, page model.Page) (model.Paged[string], error)
	IsBlacklisted(shopId uuid.UUID, name string) (bool, error)
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

func (p *ProcessorImpl) Add(shopId uuid.UUID, name string) error {
	_, err := create(p.t.Id(), shopId, name)(p.db.WithContext(p.ctx))()
	return err
}

func (p *ProcessorImpl) Remove(shopId uuid.UUID, name string) error {
	_, err := deleteByShopIdAndName(shopId, name)(p.db.WithContext(p.ctx))()
	return err
}

// NamesPaged backs the GET /merchants/{shopId}/blacklist list route
// (task-117). In-process ban checks use IsBlacklisted, not the list.
func (p *ProcessorImpl) NamesPaged(shopId uuid.UUID, page model.Page) (model.Paged[string], error) {
	ep := getByShopIdPaged(shopId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(func(e Entity) (string, error) { return e.Name, nil })(ep)(model.ParallelMap())()
}

func (p *ProcessorImpl) IsBlacklisted(shopId uuid.UUID, name string) (bool, error) {
	return existsByShopIdAndName(shopId, name)(p.db.WithContext(p.ctx))()
}
