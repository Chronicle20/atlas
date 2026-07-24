package mock

import (
	"atlas-consumables/asset"
	"atlas-consumables/consumable"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	RequestItemConsumeFunc     func(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16) error
	RequestFeedFunc            func(worldId world.Id, characterId uint32, slot int16, itemId item2.Id) error
	ConsumeErrorFunc           func(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error
	RequestScrollFunc          func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error
	RequestVegaScrollFunc      func(characterId uint32, vegaSlot int16, vegaItemId item2.Id, scrollSlot int16, equipSlot int16) error
	VegaScrollErrorFunc        func(characterId uint32, transactionId uuid.UUID, reservations []consumable.VegaReservation, err error) error
	ValidateScrollUseFunc      func(scrollItem asset.Model, equipItem asset.Model) bool
	PassScrollFunc             func(characterId uint32, legendarySpirit bool, whiteScroll bool) error
	ApplyConsumableEffectFunc  func(transactionId uuid.UUID, c channel.Model, characterId uint32, itemId item2.Id) error
	CancelConsumableEffectFunc func(transactionId uuid.UUID, characterId uint32, itemId item2.Id, f field.Model) error
	FailScrollFunc             func(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error
	RequestItemRewardFunc      func(characterId uint32, itemId item2.Id, source int16) error
	RequestViciousHammerFunc   func(characterId uint32, hammerSlot int16, equipSlot int16) error
}

var _ consumable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestItemConsume(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16) error {
	if m.RequestItemConsumeFunc != nil {
		return m.RequestItemConsumeFunc(c, characterId, slot, itemId, quantity)
	}
	return nil
}

func (m *ProcessorMock) RequestFeed(worldId world.Id, characterId uint32, slot int16, itemId item2.Id) error {
	if m.RequestFeedFunc != nil {
		return m.RequestFeedFunc(worldId, characterId, slot, itemId)
	}
	return nil
}

func (m *ProcessorMock) ConsumeError(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error {
	if m.ConsumeErrorFunc != nil {
		return m.ConsumeErrorFunc(characterId, transactionId, inventoryType, slot, err)
	}
	return nil
}

func (m *ProcessorMock) RequestScroll(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
	if m.RequestScrollFunc != nil {
		return m.RequestScrollFunc(characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit)
	}
	return nil
}

func (m *ProcessorMock) RequestVegaScroll(characterId uint32, vegaSlot int16, vegaItemId item2.Id, scrollSlot int16, equipSlot int16) error {
	if m.RequestVegaScrollFunc != nil {
		return m.RequestVegaScrollFunc(characterId, vegaSlot, vegaItemId, scrollSlot, equipSlot)
	}
	return nil
}

func (m *ProcessorMock) VegaScrollError(characterId uint32, transactionId uuid.UUID, reservations []consumable.VegaReservation, err error) error {
	if m.VegaScrollErrorFunc != nil {
		return m.VegaScrollErrorFunc(characterId, transactionId, reservations, err)
	}
	return err
}

func (m *ProcessorMock) ValidateScrollUse(scrollItem asset.Model, equipItem asset.Model) bool {
	if m.ValidateScrollUseFunc != nil {
		return m.ValidateScrollUseFunc(scrollItem, equipItem)
	}
	return false
}

func (m *ProcessorMock) PassScroll(characterId uint32, legendarySpirit bool, whiteScroll bool) error {
	if m.PassScrollFunc != nil {
		return m.PassScrollFunc(characterId, legendarySpirit, whiteScroll)
	}
	return nil
}

func (m *ProcessorMock) ApplyConsumableEffect(transactionId uuid.UUID, c channel.Model, characterId uint32, itemId item2.Id) error {
	if m.ApplyConsumableEffectFunc != nil {
		return m.ApplyConsumableEffectFunc(transactionId, c, characterId, itemId)
	}
	return nil
}

func (m *ProcessorMock) CancelConsumableEffect(transactionId uuid.UUID, characterId uint32, itemId item2.Id, f field.Model) error {
	if m.CancelConsumableEffectFunc != nil {
		return m.CancelConsumableEffectFunc(transactionId, characterId, itemId, f)
	}
	return nil
}

func (m *ProcessorMock) FailScroll(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error {
	if m.FailScrollFunc != nil {
		return m.FailScrollFunc(characterId, cursed, legendarySpirit, whiteScroll)
	}
	return nil
}

func (m *ProcessorMock) RequestViciousHammer(characterId uint32, hammerSlot int16, equipSlot int16) error {
	if m.RequestViciousHammerFunc != nil {
		return m.RequestViciousHammerFunc(characterId, hammerSlot, equipSlot)
	}
	return nil
}

func (m *ProcessorMock) RequestItemReward(characterId uint32, itemId item2.Id, source int16) error {
	if m.RequestItemRewardFunc != nil {
		return m.RequestItemRewardFunc(characterId, itemId, source)
	}
	return nil
}
