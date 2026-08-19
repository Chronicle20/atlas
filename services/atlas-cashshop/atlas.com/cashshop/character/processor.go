package character

import (
	"atlas-cashshop/character/inventory"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	InventoryDecorator(m Model) Model
	ExtendEquipSlot(characterId uint32, slotIndex int16, days uint16) (time.Time, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	ip  inventory.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		ip:  inventory.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		mp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(p.ctx, characterId), Extract)
		return model.Map(model.Decorate(decorators))(mp)()
	}
}

func (p *ProcessorImpl) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

// ExtendEquipSlot extends the character's equip-slot extension via
// atlas-character's write route (task-240 task 23, R2 -- the write side
// task 22 deferred). slotIndex is the Atlas canonical position (R1); this
// call carries it through unchanged, it does not resolve or invent it.
func (p *ProcessorImpl) ExtendEquipSlot(characterId uint32, slotIndex int16, days uint16) (time.Time, error) {
	return requests.Provider[EquipSlotExtensionRestModel, time.Time](p.l, p.ctx)(requestExtendEquipSlot(p.ctx, characterId, slotIndex, days), ExtractEquipSlotExtension)()
}
