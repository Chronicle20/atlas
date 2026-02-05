package consumable

import (
	consumable2 "atlas-channel/kafka/message/consumable"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/sirupsen/logrus"
)

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

func (p *Processor) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32) error {
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. updateTime [%d]", characterId, itemId, source, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(f, characterId, source, itemId, 1))
}

func (p *Processor) RequestScrollUse(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to scroll item in slot [%d] with scroll from slot [%d]. whiteScroll [%t], legendarySpirit [%t], updateTime [%d].", characterId, equipSlot, scrollSlot, whiteScroll, legendarySpirit, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestScrollCommandProvider(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit))
}
