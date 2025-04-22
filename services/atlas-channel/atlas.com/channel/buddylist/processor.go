package buddylist

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *Processor) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()
}

func (p *Processor) RequestAdd(characterId uint32, worldId world.Id, targetId uint32, group string) error {
	p.l.Debugf("Character [%d] would like to add [%d] to group [%s] to their buddy list.", characterId, targetId, group)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(requestAddBuddyCommandProvider(characterId, worldId, targetId, group))
}

func (p *Processor) RequestDelete(characterId uint32, worldId world.Id, targetId uint32) error {
	p.l.Debugf("Character [%d] attempting to delete buddy [%d].", characterId, targetId)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(requestDeleteBuddyCommandProvider(characterId, worldId, targetId))
}
