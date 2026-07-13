package consumable

import (
	consumable2 "atlas-channel/kafka/message/consumable"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32) error
	RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error
	RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error
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

func (p *ProcessorImpl) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32) error {
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. updateTime [%d]", characterId, itemId, source, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(f, characterId, source, itemId, 1))
}

func (p *ProcessorImpl) RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to scroll item in slot [%d] with scroll from slot [%d]. whiteScroll [%t], legendarySpirit [%t], updateTime [%d].", characterId, equipSlot, scrollSlot, whiteScroll, legendarySpirit, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestScrollCommandProvider(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit))
}

func (p *ProcessorImpl) RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error {
	p.l.Debugf("Character [%d] attempting to use vicious hammer in slot [%d] on equip slot [%d].", characterId, hammerSlot, equipSlot)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestViciousHammerCommandProvider(f, characterId, hammerSlot, equipSlot))
}
