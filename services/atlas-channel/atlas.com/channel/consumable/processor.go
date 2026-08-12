package consumable

import (
	consumable2 "atlas-channel/kafka/message/consumable"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Processor interface {
	RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error
	RequestItemConsumeWithPet(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error
	RequestItemReward(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error
	RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error
	RequestVegaScrollUse(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error
	RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error
	RequestSkillBookUse(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id, updateTime uint32) error
	RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error
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

func (p *ProcessorImpl) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	// Defense in depth for an absent/"0"-string itemConNo (FR-1): an absent
	// amount means one item, never zero.
	if quantity < 1 {
		quantity = 1
	}
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. quantity [%d], updateTime [%d]", characterId, itemId, source, quantity, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(f, characterId, source, itemId, quantity))
}

// RequestItemConsumeWithPet is RequestItemConsume for consume paths that carry
// a target pet (0519 pet skill pouches). The auto-pot path deliberately does
// NOT use it: its pet validation happens at the socket handler and nothing
// downstream needs the pet.
func (p *ProcessorImpl) RequestItemConsumeWithPet(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error {
	p.l.Debugf("Character [%d] using pet skill item [%d] from slot [%d] on pet [%d]. updateTime [%d]", characterId, itemId, source, petId, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeWithPetCommandProvider(f, characterId, source, itemId, 1, petId))
}

func (p *ProcessorImpl) RequestItemReward(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error {
	p.l.Debugf("Character [%d] using reward box [%d] from slot [%d].", characterId, itemId, source)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemRewardCommandProvider(f, characterId, source, itemId))
}

func (p *ProcessorImpl) RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to scroll item in slot [%d] with scroll from slot [%d]. whiteScroll [%t], legendarySpirit [%t], updateTime [%d].", characterId, equipSlot, scrollSlot, whiteScroll, legendarySpirit, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestScrollCommandProvider(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit))
}

func (p *ProcessorImpl) RequestVegaScrollUse(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error {
	p.l.Debugf("Character [%d] attempting vega scroll [%d] from cash slot [%d]: scroll slot [%d] onto equip slot [%d].", characterId, vegaItemId, vegaSlot, scrollSlot, equipSlot)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestVegaScrollCommandProvider(f, characterId, vegaSlot, vegaItemId, scrollSlot, equipSlot))
}

func (p *ProcessorImpl) RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error {
	p.l.Debugf("Character [%d] attempting to use vicious hammer in slot [%d] on equip slot [%d].", characterId, hammerSlot, equipSlot)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestViciousHammerCommandProvider(f, characterId, hammerSlot, equipSlot))
}

func (p *ProcessorImpl) RequestSkillBookUse(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id, updateTime uint32) error {
	p.l.Debugf("Character [%d] using skill book [%d] from slot [%d]. updateTime [%d]", characterId, itemId, slot, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestSkillBookUseCommandProvider(f, characterId, slot, itemId))
}

func (p *ProcessorImpl) RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error {
	p.l.Debugf("Character [%d] using catch item [%d] from slot [%d] on monster [%d].", characterId, itemId, source, monsterUniqueId)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestCatchMonsterCommandProvider(f, characterId, source, itemId, monsterUniqueId))
}
