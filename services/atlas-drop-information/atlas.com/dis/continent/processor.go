package continent

import (
	"atlas-drops-information/continent/drop"
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
	return func() ([]Model, error) {
		ms := make(map[int32]Model)
		drops, err := drop.NewProcessor(p.l, p.ctx, p.db).GetAll()()
		if err != nil {
			return nil, err
		}

		for _, d := range drops {
			if _, ok := ms[d.ContinentId()]; !ok {
				m := Model{
					id:    d.ContinentId(),
					drops: make([]drop.Model, 0),
				}
				ms[d.ContinentId()] = m
			}
			m := ms[d.ContinentId()]
			m.drops = append(m.drops, d)
			ms[d.ContinentId()] = m
		}

		results := make([]Model, 0)
		for _, m := range ms {
			results = append(results, m)
		}
		return results, nil
	}
}
