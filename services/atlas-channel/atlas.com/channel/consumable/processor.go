package consumable

import (
	consumable2 "atlas-channel/kafka/message/consumable"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
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

func (p *Processor) RequestItemConsume(worldId world.Id, channelId channel.Id, characterId uint32, itemId uint32, slot int16, updateTime uint32) error {
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. updateTime [%d]", characterId, itemId, slot, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(worldId, channelId, characterId, slot, itemId, 1))
}

func (p *Processor) RequestScrollUse(worldId world.Id, channelId channel.Id, characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to scroll item in slot [%d] with scroll from slot [%d]. whiteScroll [%t], legendarySpirit [%t], updateTime [%d].", characterId, equipSlot, scrollSlot, whiteScroll, legendarySpirit, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestScrollCommandProvider(worldId, channelId, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit))
}
