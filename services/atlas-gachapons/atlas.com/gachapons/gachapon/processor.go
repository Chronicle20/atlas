package gachapon

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) error
	Update(id string, name string, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error
	Delete(id string) error
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

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll(p.t.Id())(p.db))()
}

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	e, err := getById(p.t.Id(), id)(p.db)()
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateGachapon(p.db, m)
}

func (p *ProcessorImpl) Update(id string, name string, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	return UpdateGachapon(p.db, p.t.Id(), id, name, commonWeight, uncommonWeight, rareWeight)
}

func (p *ProcessorImpl) Delete(id string) error {
	return DeleteGachapon(p.db, p.t.Id(), id)
}
