package item

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByGachaponId(gachaponId string) model.Provider[[]Model]
	GetByGachaponIdAndTier(gachaponId string, tier string) model.Provider[[]Model]
	Create(m Model) error
	Delete(id uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: t}
}

func (p *ProcessorImpl) GetByGachaponId(gachaponId string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponId(p.t.Id(), gachaponId)(p.db))()
}

func (p *ProcessorImpl) GetByGachaponIdAndTier(gachaponId string, tier string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponIdAndTier(p.t.Id(), gachaponId, tier)(p.db))()
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateItem(p.db, m)
}

func (p *ProcessorImpl) Delete(id uint32) error {
	return DeleteItem(p.db, p.t.Id(), id)
}
