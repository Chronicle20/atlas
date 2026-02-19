package commodities

import (
	"atlas-npc/data/consumable"
	"atlas-npc/data/etc"
	"atlas-npc/data/setup"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-model/model"
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
	CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	DeleteCommodity(id uuid.UUID) error
	DeleteAllCommoditiesByNpcId(npcId uint32) error
	DeleteAllCommodities() error
	WithTransaction(tx *gorm.DB) Processor
	ExistsByNpcId(npcId uint32) (bool, error)
	GetDistinctNpcIds() ([]uint32, error)
	DistinctNpcIdsProvider() model.Provider[[]uint32]
}

type ProcessorImpl struct {
	l                logrus.FieldLogger
	ctx              context.Context
	db               *gorm.DB
	GetByNpcIdFn     func(npcId uint32) ([]Model, error)
	GetAllByTenantFn func() ([]Model, error)
	CreateFn         func(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	UpdateFn         func(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error)
	DeleteFn         func(id uuid.UUID) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	newProcessor := &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
	}
	newProcessor.GetByNpcIdFn = p.GetByNpcIdFn
	newProcessor.GetAllByTenantFn = p.GetAllByTenantFn
	newProcessor.CreateFn = p.CreateFn
	newProcessor.UpdateFn = p.UpdateFn
	newProcessor.DeleteFn = p.DeleteFn
	return newProcessor
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) ([]Model, error) {
	if p.GetByNpcIdFn != nil {
		return p.GetByNpcIdFn(npcId)
	}
	return p.ByNpcIdProvider(npcId)()
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	mp := model.SliceMap(Make)(getByNpcId(npcId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
	return model.SliceMap(model.Decorate(model.Decorators(p.DataDecorator)))(mp)(model.ParallelMap())
}

func (p *ProcessorImpl) DataDecorator(m Model) Model {
	b := Clone(m)

	// Determine the inventory type from the templateId
	it, ok := inventory.TypeFromItemId(item.Id(m.TemplateId()))
	if !ok {
		result, err := b.Build()
		if err != nil {
			return m
		}
		return result
	}

	if it == inventory.TypeValueEquip {
		b.SetUnitPrice(1)
		b.SetSlotMax(1)
	} else if it == inventory.TypeValueUse {
		// For consumable, get unitPrice and slotMax from the model
		cm, err := consumable.NewProcessor(p.l, p.ctx).GetById(m.TemplateId())
		if err == nil {
			b.SetUnitPrice(cm.UnitPrice())
			b.SetSlotMax(cm.SlotMax())
		}
	} else if it == inventory.TypeValueSetup {
		sm, err := setup.NewProcessor(p.l, p.ctx).GetById(m.TemplateId())
		if err == nil {
			b.SetUnitPrice(1)
			b.SetSlotMax(sm.SlotMax())
		}
	} else if it == inventory.TypeValueETC {
		em, err := etc.NewProcessor(p.l, p.ctx).GetById(m.TemplateId())
		if err == nil {
			b.SetUnitPrice(em.UnitPrice())
			b.SetSlotMax(em.SlotMax())
		}
	}
	result, err := b.Build()
	if err != nil {
		return m
	}
	return result
}

func (p *ProcessorImpl) CreateCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	if p.CreateFn != nil {
		return p.CreateFn(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	}
	c, err := createCommodity(p.ctx, p.db.WithContext(p.ctx))(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		return Model{}, err
	}
	return model.Map(model.Decorate(model.Decorators(p.DataDecorator)))(model.FixedProvider(c))()
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	if p.UpdateFn != nil {
		return p.UpdateFn(id, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)

	}
	c, err := updateCommodity(p.ctx, p.db.WithContext(p.ctx))(id, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		return Model{}, err
	}
	return model.Map(model.Decorate(model.Decorators(p.DataDecorator)))(model.FixedProvider(c))()
}

func (p *ProcessorImpl) DeleteCommodity(id uuid.UUID) error {
	if p.DeleteFn != nil {
		return p.DeleteFn(id)
	}
	return deleteCommodity(p.ctx, p.db.WithContext(p.ctx))(id)
}

func (p *ProcessorImpl) GetAllByTenant() ([]Model, error) {
	if p.GetAllByTenantFn != nil {
		return p.GetAllByTenantFn()
	}
	return p.ByTenantProvider()()
}

func (p *ProcessorImpl) ByTenantProvider() model.Provider[[]Model] {
	mp := model.SliceMap(Make)(getAllByTenant()(p.db.WithContext(p.ctx)))(model.ParallelMap())
	return model.SliceMap(model.Decorate(model.Decorators(p.DataDecorator)))(mp)(model.ParallelMap())
}

func (p *ProcessorImpl) GetCommodityIdToNpcIdMap() (map[uuid.UUID]uint32, error) {
	return p.CommodityIdToNpcIdMapProvider()()
}

func (p *ProcessorImpl) CommodityIdToNpcIdMapProvider() model.Provider[map[uuid.UUID]uint32] {
	return getCommodityIdToNpcIdMap()(p.db.WithContext(p.ctx))
}

func (p *ProcessorImpl) DeleteAllCommoditiesByNpcId(npcId uint32) error {
	return deleteAllCommoditiesByNpcId(p.ctx, p.db.WithContext(p.ctx))(npcId)
}

func (p *ProcessorImpl) DeleteAllCommodities() error {
	return deleteAllCommodities(p.ctx, p.db.WithContext(p.ctx))()
}

func (p *ProcessorImpl) ExistsByNpcId(npcId uint32) (bool, error) {
	return existsByNpcId(npcId)(p.db.WithContext(p.ctx))()
}

func (p *ProcessorImpl) GetDistinctNpcIds() ([]uint32, error) {
	return p.DistinctNpcIdsProvider()()
}

func (p *ProcessorImpl) DistinctNpcIdsProvider() model.Provider[[]uint32] {
	return getDistinctNpcIds()(p.db.WithContext(p.ctx))
}
