package mock

import (
	"atlas-inventory/kafka/message"

	"github.com/Chronicle20/atlas-constants/field"
)

type ProcessorImpl struct {
	CreateForEquipmentFn func(mb *message.Buffer) func(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CreateForItemFn      func(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error
	CancelReservationFn  func(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
	RequestPickUpFn      func(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error
}

func (p *ProcessorImpl) CreateForEquipment(mb *message.Buffer) func(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return p.CreateForEquipmentFn(mb)
}

func (p *ProcessorImpl) CreateForItem(mb *message.Buffer) func(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return p.CreateForItemFn(mb)
}

func (p *ProcessorImpl) CancelReservation(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return p.CancelReservationFn(mb)
}

func (p *ProcessorImpl) RequestPickUp(mb *message.Buffer) func(f field.Model, dropId uint32, characterId uint32) error {
	return p.RequestPickUpFn(mb)
}
