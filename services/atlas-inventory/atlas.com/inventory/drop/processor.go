package drop

import (
	"atlas-inventory/kafka/message"
	dropMsg "atlas-inventory/kafka/message/drop"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	CreateForEquipment(mb *message.Buffer) func(f field.Model, itemId uint32, ed dropMsg.EquipmentData, dropType byte, x int16, y int16, ownerId uint32) error
	CreateForItem(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CancelReservation(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
	RequestPickUp(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CreateForEquipment(mb *message.Buffer) func(f field.Model, itemId uint32, ed dropMsg.EquipmentData, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(f field.Model, itemId uint32, ed dropMsg.EquipmentData, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(dropMsg.EnvCommandTopic, EquipmentProvider(f, itemId, ed, dropType, x, y, ownerId))
	}
}

func (p *ProcessorImpl) CreateForItem(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(dropMsg.EnvCommandTopic, ItemProvider(f, itemId, quantity, dropType, x, y, ownerId))
	}
}

func (p *ProcessorImpl) CancelReservation(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return func(f field.Model, dropId uint32, characterId uint32) error {
		return mb.Put(dropMsg.EnvCommandTopic, CancelReservationCommandProvider(f, dropId, characterId))
	}
}

func (p *ProcessorImpl) RequestPickUp(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return func(f field.Model, dropId uint32, characterId uint32) error {
		return mb.Put(dropMsg.EnvCommandTopic, RequestPickUpCommandProvider(f, dropId, characterId))
	}
}
