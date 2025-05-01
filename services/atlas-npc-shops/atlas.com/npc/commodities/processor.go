package commodities

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) ([]Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[[]Model]
	CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error)
	DeleteCommodity(id uuid.UUID) error
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	GetByNpcIdFn func(npcId uint32) ([]Model, error)
	createFn     func(npcId uint32, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error)
	updateFn     func(id uuid.UUID, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error)
	deleteFn     func(id uuid.UUID) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		createFn: createCommodity(ctx, db),
		updateFn: updateCommodity(ctx, db),
		deleteFn: deleteCommodity(ctx, db),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) ([]Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByNpcId(p.t.Id(), npcId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error) {
	return p.createFn(npcId, templateId, mesoPrice, perfectPitchPrice)
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, perfectPitchPrice uint32) (Model, error) {
	return p.updateFn(id, templateId, mesoPrice, perfectPitchPrice)
}

func (p *ProcessorImpl) DeleteCommodity(id uuid.UUID) error {
	return p.deleteFn(id)
}
