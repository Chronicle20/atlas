package shops

import (
	"atlas-npc/commodities"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) (Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[Model]
	AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	RemoveCommodity(id uuid.UUID) error
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	GetByNpcIdFn func(npcId uint32) (Model, error)
	cp           commodities.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		cp:  commodities.NewProcessor(l, ctx, db),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) (Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	cms, err := p.cp.GetByNpcId(npcId)
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(NewBuilder(npcId).SetCommodities(cms).Build())
}

func (p *ProcessorImpl) AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.UpdateCommodity(id, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) RemoveCommodity(id uuid.UUID) error {
	return p.cp.DeleteCommodity(id)
}
