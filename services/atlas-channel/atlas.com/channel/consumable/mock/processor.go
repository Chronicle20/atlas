package mock

import (
	"atlas-channel/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	RequestItemConsumeFunc        func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error
	RequestItemConsumeWithPetFunc func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error
	RequestItemRewardFunc         func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error
	RequestScrollUseFunc          func(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error
	RequestVegaScrollUseFunc      func(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error
	RequestViciousHammerUseFunc   func(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error
	RequestSkillBookUseFunc       func(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id, updateTime uint32) error
	RequestCatchMonsterFunc       func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error
}

var _ consumable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	if m.RequestItemConsumeFunc != nil {
		return m.RequestItemConsumeFunc(f, characterId, itemId, source, quantity, updateTime)
	}
	return nil
}

func (m *ProcessorMock) RequestItemConsumeWithPet(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error {
	if m.RequestItemConsumeWithPetFunc != nil {
		return m.RequestItemConsumeWithPetFunc(f, characterId, itemId, source, updateTime, petId)
	}
	return nil
}

func (m *ProcessorMock) RequestItemReward(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error {
	if m.RequestItemRewardFunc != nil {
		return m.RequestItemRewardFunc(f, characterId, itemId, source)
	}
	return nil
}

func (m *ProcessorMock) RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	if m.RequestScrollUseFunc != nil {
		return m.RequestScrollUseFunc(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit, updateTime)
	}
	return nil
}

func (m *ProcessorMock) RequestVegaScrollUse(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error {
	if m.RequestVegaScrollUseFunc != nil {
		return m.RequestVegaScrollUseFunc(f, characterId, vegaItemId, vegaSlot, scrollSlot, equipSlot)
	}
	return nil
}

func (m *ProcessorMock) RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error {
	if m.RequestViciousHammerUseFunc != nil {
		return m.RequestViciousHammerUseFunc(f, characterId, hammerSlot, equipSlot)
	}
	return nil
}

func (m *ProcessorMock) RequestSkillBookUse(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id, updateTime uint32) error {
	if m.RequestSkillBookUseFunc != nil {
		return m.RequestSkillBookUseFunc(f, characterId, slot, itemId, updateTime)
	}
	return nil
}

func (m *ProcessorMock) RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error {
	if m.RequestCatchMonsterFunc != nil {
		return m.RequestCatchMonsterFunc(f, characterId, itemId, source, monsterUniqueId)
	}
	return nil
}
