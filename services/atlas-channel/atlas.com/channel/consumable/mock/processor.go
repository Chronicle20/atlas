package mock

import (
	"atlas-channel/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	RequestItemConsumeFunc      func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32) error
	RequestItemRewardFunc       func(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error
	RequestScrollUseFunc        func(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error
	RequestVegaScrollUseFunc    func(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error
	RequestViciousHammerUseFunc func(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error
}

var _ consumable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32) error {
	if m.RequestItemConsumeFunc != nil {
		return m.RequestItemConsumeFunc(f, characterId, itemId, source, updateTime)
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
