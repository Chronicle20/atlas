package drop

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
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

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}
