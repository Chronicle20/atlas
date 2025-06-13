package compartment

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/character/compartment"
	compartment2 "atlas-cashshop/kafka/producer/character/compartment"
	"context"
	inventory3 "github.com/Chronicle20/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	IncreaseCapacity(mb *message.Buffer) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error
	MoveCashItemToCompartment(mb *message.Buffer) func(characterId uint32, inventoryType byte, slot int16, cashItemId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *ProcessorImpl) IncreaseCapacity(mb *message.Buffer) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
	return func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
		return mb.Put(compartment.EnvCommandTopic, compartment2.IncreaseCapacityCommandProvider(characterId, byte(inventoryType), amount))
	}
}

func (p *ProcessorImpl) MoveCashItemToCompartment(mb *message.Buffer) func(characterId uint32, inventoryType byte, slot int16, cashItemId uint32) error {
	return func(characterId uint32, inventoryType byte, slot int16, cashItemId uint32) error {
		p.l.Debugf("Character [%d] taking ownership of cash item [%d] in inventory type [%d] slot [%d].", characterId, cashItemId, inventoryType, slot)

		// Send a command to the inventory service to add the item to the character's inventory
		return mb.Put(compartment.EnvCommandTopic, compartment2.MoveCashItemCommandProvider(characterId, inventoryType, slot, cashItemId))
	}
}
