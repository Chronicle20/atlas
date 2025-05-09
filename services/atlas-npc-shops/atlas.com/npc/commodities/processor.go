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
	GetAllByTenant() ([]Model, error)
	ByTenantProvider() model.Provider[[]Model]
	GetCommodityIdToNpcIdMap() (map[uuid.UUID]uint32, error)
	CommodityIdToNpcIdMapProvider() model.Provider[map[uuid.UUID]uint32]
	CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	DeleteCommodity(id uuid.UUID) error
}

type ProcessorImpl struct {
	l                logrus.FieldLogger
	ctx              context.Context
	db               *gorm.DB
	t                tenant.Model
	GetByNpcIdFn     func(npcId uint32) ([]Model, error)
	GetAllByTenantFn func() ([]Model, error)
	CreateFn         func(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	UpdateFn         func(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	DeleteFn         func(id uuid.UUID) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		CreateFn: createCommodity(ctx, db),
		UpdateFn: updateCommodity(ctx, db),
		DeleteFn: deleteCommodity(ctx, db),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	p.GetAllByTenantFn = func() ([]Model, error) {
		return p.ByTenantProvider()()
	}
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) ([]Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByNpcId(p.t.Id(), npcId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	return p.CreateFn(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	return p.UpdateFn(id, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) DeleteCommodity(id uuid.UUID) error {
	return p.DeleteFn(id)
}

func (p *ProcessorImpl) GetAllByTenant() ([]Model, error) {
	return p.GetAllByTenantFn()
}

func (p *ProcessorImpl) ByTenantProvider() model.Provider[[]Model] {
	return model.SliceMap(Make)(getAllByTenant(p.t.Id())(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetCommodityIdToNpcIdMap() (map[uuid.UUID]uint32, error) {
	return p.CommodityIdToNpcIdMapProvider()()
}

func (p *ProcessorImpl) CommodityIdToNpcIdMapProvider() model.Provider[map[uuid.UUID]uint32] {
	return getCommodityIdToNpcIdMap(p.t.Id())(p.db)
}
