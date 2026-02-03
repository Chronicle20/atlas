package drop

import (
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/drop"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Provider interface {
	CreateForEquipment(mb *message.Buffer) func(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CreateForItem(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CancelReservation(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
	RequestPickUp(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
}

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) CreateForEquipment(mb *message.Buffer) func(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(drop.EnvCommandTopic, EquipmentProvider(f, itemId, equipmentId, dropType, x, y, ownerId))
	}
}

func (p *Processor) CreateForItem(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(drop.EnvCommandTopic, ItemProvider(f, itemId, quantity, dropType, x, y, ownerId))
	}
}

func (p *Processor) CancelReservation(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return func(f field.Model, dropId uint32, characterId uint32) error {
		return mb.Put(drop.EnvCommandTopic, CancelReservationCommandProvider(f, dropId, characterId))
	}
}

func (p *Processor) RequestPickUp(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return func(f field.Model, dropId uint32, characterId uint32) error {
		return mb.Put(drop.EnvCommandTopic, RequestPickUpCommandProvider(f, dropId, characterId))
	}
}
