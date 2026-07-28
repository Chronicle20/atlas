package searchcount

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	RecordSearch(worldId world.Id, itemId uint32) error
	GetTop(worldId world.Id, limit int) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) RecordSearch(worldId world.Id, itemId uint32) error {
	return incrementSearchCount(p.t.Id(), worldId, itemId)(p.db.WithContext(p.ctx))
}

func (p *ProcessorImpl) GetTop(worldId world.Id, limit int) ([]Model, error) {
	return model.SliceMap(Make)(getTopByWorld(worldId, limit)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
