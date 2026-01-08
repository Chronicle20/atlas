package drop

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
	GetForMonster(monsterId uint32) model.Provider[[]Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   t,
	}
}

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(makeDrop)(getAll(p.t.Id())(p.db))()
}

func (p *ProcessorImpl) GetForMonster(monsterId uint32) model.Provider[[]Model] {
	return model.SliceMap(makeDrop)(getByMonsterId(p.t.Id(), monsterId)(p.db))()
}

// Legacy function wrappers for backward compatibility during migration
func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
		return func(db *gorm.DB) ([]Model, error) {
			return NewProcessor(l, ctx, db).GetAll()()
		}
	}
}

func GetForMonster(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
		return func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
			return func(monsterId uint32) ([]Model, error) {
				return NewProcessor(l, ctx, db).GetForMonster(monsterId)()
			}
		}
	}
}
