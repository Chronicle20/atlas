package drop

import (
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/drop"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Provider interface {
	CreateForEquipment(mb *message.Buffer) func(m _map.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CreateForItem(mb *message.Buffer) func(m _map.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CancelReservation(mb *message.Buffer) func(m _map.Model, dropId uint32, characterId uint32) error
	RequestPickUp(mb *message.Buffer) func(m _map.Model, dropId uint32, characterId uint32) error
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

func (p *Processor) CreateForEquipment(mb *message.Buffer) func(m _map.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(m _map.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(drop.EnvCommandTopic, EquipmentProvider(m, itemId, equipmentId, dropType, x, y, ownerId))
	}
}

func (p *Processor) CreateForItem(mb *message.Buffer) func(m _map.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(m _map.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return mb.Put(drop.EnvCommandTopic, ItemProvider(m, itemId, quantity, dropType, x, y, ownerId))
	}
}

func (p *Processor) CancelReservation(mb *message.Buffer) func(m _map.Model, dropId uint32, characterId uint32) error {
	return func(m _map.Model, dropId uint32, characterId uint32) error {
		return mb.Put(drop.EnvCommandTopic, CancelReservationCommandProvider(m, dropId, characterId))
	}
}

func (p *Processor) RequestPickUp(mb *message.Buffer) func(m _map.Model, dropId uint32, characterId uint32) error {
	return func(m _map.Model, dropId uint32, characterId uint32) error {
		return mb.Put(drop.EnvCommandTopic, RequestPickUpCommandProvider(m, dropId, characterId))
	}
}
