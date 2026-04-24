package gachapon

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) error
	Update(id string, name string, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error
	Delete(id string) error
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

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	e, err := getById(id)(p.db.WithContext(p.ctx))()
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

func (p *ProcessorImpl) Create(m Model) error {
	return CreateGachapon(p.db.WithContext(p.ctx), m)
}

func (p *ProcessorImpl) Update(id string, name string, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	return UpdateGachapon(p.db.WithContext(p.ctx), id, name, commonWeight, uncommonWeight, rareWeight)
}

func (p *ProcessorImpl) Delete(id string) error {
	return DeleteGachapon(p.db.WithContext(p.ctx), id)
}

func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	// gachapons has no updated_at column; updatedAt is always nil.
	return count, nil, nil
}
