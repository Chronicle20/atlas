package global

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
	GetByTier(tier string) model.Provider[[]Model]
	Create(m Model) error
	Delete(id uint32) error
	Count() (int64, *time.Time, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByTier(tier string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByTier(tier)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateItem(p.db.WithContext(p.ctx), m)
}

func (p *ProcessorImpl) Delete(id uint32) error {
	return DeleteItem(p.db.WithContext(p.ctx), id)
}

func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	// global_gachapon_items has no updated_at column; updatedAt is always nil.
	return count, nil, nil
}
