package drop

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// GetAll returns every monster drop for the tenant, unpaged. Kept for
	// continent.Processor.GetAll's semantic-all aggregation (it groups every
	// drop by continent, so it needs the complete set, not one page of it).
	GetAll() model.Provider[[]Model]
	GetForMonster(monsterId uint32, page model.Page) model.Provider[model.Paged[Model]]
	GetForItem(itemId uint32, page model.Page) model.Provider[model.Paged[Model]]
	Count() (int64, *time.Time, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetForMonster(monsterId uint32, page model.Page) model.Provider[model.Paged[Model]] {
	ep := getByMonsterIdPagedProvider(monsterId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) GetForItem(itemId uint32, page model.Page) model.Provider[model.Paged[Model]] {
	ep := getByItemIdPagedProvider(itemId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	// monster_drops has no updated_at column; updatedAt is always nil.
	return count, nil, nil
}
