package food

import (
	"atlas-channel/kafka/message/food"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestFeed(f field.Model, characterId character.Id, slot int16, itemId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// RequestFeed emits a taming-mob (mount) food command to consumables. It performs
// no item mutation; consumables decrements the item (Task 32). The field carries
// worldId/channelId/mapId/instance so the eventual fed event can be routed.
func (p *ProcessorImpl) RequestFeed(f field.Model, characterId character.Id, slot int16, itemId uint32) error {
	p.l.Debugf("Character [%d] feeding mount with item [%d] from slot [%d].", characterId, itemId, slot)
	return producer.ProviderImpl(p.l)(p.ctx)(food.EnvCommandTopic)(RequestFeedCommandProvider(f, characterId, slot, itemId))
}
