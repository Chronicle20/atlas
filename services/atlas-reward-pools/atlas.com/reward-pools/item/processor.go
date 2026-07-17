package item

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// GetByGachaponId returns every item for the given gachapon, unpaged and
	// regardless of tier. Modeled on GetByGachaponIdAndTier, minus the tier
	// filter — for weighted (e.g. incubator) reward pools that roll across
	// the whole machine rather than one tier at a time.
	GetByGachaponId(gachaponId string) model.Provider[[]Model]
	// GetByGachaponIdPaged is the resource-handler-facing paged sibling of
	// GetByGachaponId.
	GetByGachaponIdPaged(gachaponId string, page model.Page) model.Provider[model.Paged[Model]]
	// GetByGachaponIdAndTier returns every item for the given gachapon+tier,
	// unpaged. Kept for reward.getMergedPool's semantic-all read (the
	// reward-selection roulette needs the complete pool, not one page of it).
	GetByGachaponIdAndTier(gachaponId string, tier string) model.Provider[[]Model]
	// GetByGachaponIdAndTierPaged is the resource-handler-facing paged
	// sibling of GetByGachaponIdAndTier.
	GetByGachaponIdAndTierPaged(gachaponId string, tier string, page model.Page) model.Provider[model.Paged[Model]]
	Create(m Model) error
	// Update rewrites an item's itemId/quantity/tier/weight in place. The
	// owning gachapon is never re-parented.
	Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error
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

func (p *ProcessorImpl) GetByGachaponId(gachaponId string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponId(gachaponId)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByGachaponIdPaged(gachaponId string, page model.Page) model.Provider[model.Paged[Model]] {
	ep := getByGachaponIdPagedProvider(gachaponId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByGachaponIdAndTier(gachaponId string, tier string) model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getByGachaponIdAndTier(gachaponId, tier)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByGachaponIdAndTierPaged(gachaponId string, tier string, page model.Page) model.Provider[model.Paged[Model]] {
	ep := getByGachaponIdAndTierPagedProvider(gachaponId, tier, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateItem(p.db.WithContext(p.ctx), m)
}

func (p *ProcessorImpl) Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error {
	return UpdateItem(p.db.WithContext(p.ctx), id, itemId, quantity, tier, weight)
}

func (p *ProcessorImpl) Delete(id uint32) error {
	return DeleteItem(p.db.WithContext(p.ctx), id)
}

func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	// gachapon_items has no updated_at column; updatedAt is always nil.
	return count, nil, nil
}
