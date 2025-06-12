package item

import (
	"context"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

type Processor interface {
	ByIdProvider(itemId uint32) model.Provider[[]Model]
	GetById(itemId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *ProcessorImpl) ByIdProvider(itemId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byIdEntityProvider(p.t.Id(), itemId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetById(itemId uint32) ([]Model, error) {
	return p.ByIdProvider(itemId)()
}
