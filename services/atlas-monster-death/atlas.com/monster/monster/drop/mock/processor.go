package mock

import (
	"atlas-monster-death/monster/drop"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetByMonsterIdFunc func(monsterId uint32) ([]drop.Model, error)
	CreateFunc         func(f field.Model, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m drop.Model, mesoRate float64, ownerPartyId uint32) error
	SpawnMesoFunc      func(f field.Model, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m drop.Model, posX int16, posY int16, mesoRate float64, ownerPartyId uint32) error
	SpawnItemFunc      func(f field.Model, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m drop.Model, posX int16, posY int16, ownerPartyId uint32) error
	SpawnDropFunc      func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte, ed drop.EquipmentData, ownerPartyId uint32) error
}

var _ drop.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByMonsterId(monsterId uint32) ([]drop.Model, error) {
	if m.GetByMonsterIdFunc != nil {
		return m.GetByMonsterIdFunc(monsterId)
	}
	return nil, nil
}

func (m *ProcessorMock) Create(f field.Model, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, dm drop.Model, mesoRate float64, ownerPartyId uint32) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(f, index, monsterId, x, y, killerId, dropType, dm, mesoRate, ownerPartyId)
	}
	return nil
}

func (m *ProcessorMock) SpawnMeso(f field.Model, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, dm drop.Model, posX int16, posY int16, mesoRate float64, ownerPartyId uint32) error {
	if m.SpawnMesoFunc != nil {
		return m.SpawnMesoFunc(f, monsterId, x, y, killerId, dropType, dm, posX, posY, mesoRate, ownerPartyId)
	}
	return nil
}

func (m *ProcessorMock) SpawnItem(f field.Model, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, dm drop.Model, posX int16, posY int16, ownerPartyId uint32) error {
	if m.SpawnItemFunc != nil {
		return m.SpawnItemFunc(f, itemId, monsterId, x, y, killerId, dropType, dm, posX, posY, ownerPartyId)
	}
	return nil
}

func (m *ProcessorMock) SpawnDrop(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte, ed drop.EquipmentData, ownerPartyId uint32) error {
	if m.SpawnDropFunc != nil {
		return m.SpawnDropFunc(f, itemId, quantity, mesos, posX, posY, monsterX, monsterY, monsterId, killerId, playerDrop, dropType, ed, ownerPartyId)
	}
	return nil
}
