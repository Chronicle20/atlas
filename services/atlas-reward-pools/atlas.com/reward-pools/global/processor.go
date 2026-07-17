package global

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll(page model.Page) model.Provider[model.Paged[Model]]
	// GetByTier returns every global item for the given tier, unpaged. Kept
	// for reward.getMergedPool's semantic-all read (the reward-selection
	// roulette needs the complete pool, not one page of it).
	GetByTier(tier string) model.Provider[[]Model]
	// GetByTierPaged is the resource-handler-facing paged sibling of
	// GetByTier.
	GetByTierPaged(tier string, page model.Page) model.Provider[model.Paged[Model]]
	Create(m Model) error
	Update(id uint32, itemId uint32, quantity uint32, tier string) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAll(page model.Page) model.Provider[model.Paged[Model]] {
	ep := getAllPagedProvider(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByTier(tier string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByTier(tier)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByTierPaged(tier string, page model.Page) model.Provider[model.Paged[Model]] {
	ep := getByTierPagedProvider(tier, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateItem(p.db.WithContext(p.ctx), m)
}

func (p *ProcessorImpl) Update(id uint32, itemId uint32, quantity uint32, tier string) error {
	if !isValidTier(tier) {
		return ErrInvalidTier
	}
	return UpdateItem(p.db.WithContext(p.ctx), id, itemId, quantity, tier)
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
