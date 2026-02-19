package item

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
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
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

func (p *ProcessorImpl) GetByGachaponId(gachaponId string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponId(gachaponId)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByGachaponIdAndTier(gachaponId string, tier string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponIdAndTier(gachaponId, tier)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateItem(p.db.WithContext(p.ctx), m)
}

func (p *ProcessorImpl) Delete(id uint32) error {
	return DeleteItem(p.db.WithContext(p.ctx), id)
}
