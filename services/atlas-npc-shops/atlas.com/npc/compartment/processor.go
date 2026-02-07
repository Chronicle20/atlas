package compartment

import (
	"atlas-npc/kafka/message"
	compartmentMessage "atlas-npc/kafka/message/compartment"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
)

type Processor interface {
	RequestCreateItem(mb *message.Buffer) func(characterId uint32, templateId uint32, quantity uint32) error
	RequestDestroyItem(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	RequestRechargeItem(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
}

type ProcessorImpl struct{}

func NewProcessor() Processor {
	return &ProcessorImpl{}
}

func (p *ProcessorImpl) RequestCreateItem(mb *message.Buffer) func(characterId uint32, templateId uint32, quantity uint32) error {
	return func(characterId uint32, templateId uint32, quantity uint32) error {
		inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
		if !ok {
			return errors.New("invalid templateId")
		}
		return mb.Put(compartmentMessage.EnvCommandTopic, RequestCreateAssetCommandProvider(characterId, inventoryType, templateId, quantity))
	}
}

func (p *ProcessorImpl) RequestDestroyItem(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		return mb.Put(compartmentMessage.EnvCommandTopic, RequestDestroyAssetCommandProvider(characterId, inventoryType, slot, quantity))
	}
}

func (p *ProcessorImpl) RequestRechargeItem(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		return mb.Put(compartmentMessage.EnvCommandTopic, RequestRechargeAssetCommandProvider(characterId, inventoryType, slot, quantity))
	}
}
