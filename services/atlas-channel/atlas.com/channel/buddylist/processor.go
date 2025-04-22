package buddylist

import (
	buddylist2 "atlas-channel/kafka/message/buddylist"
	"atlas-channel/kafka/producer"
	buddylist3 "atlas-channel/kafka/producer/buddylist"
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
	return producer.ProviderImpl(p.l)(p.ctx)(buddylist2.EnvCommandTopic)(buddylist3.RequestAddBuddyCommandProvider(characterId, worldId, targetId, group))
}

func (p *Processor) RequestDelete(characterId uint32, worldId world.Id, targetId uint32) error {
	p.l.Debugf("Character [%d] attempting to delete buddy [%d].", characterId, targetId)
	return producer.ProviderImpl(p.l)(p.ctx)(buddylist2.EnvCommandTopic)(buddylist3.RequestDeleteBuddyCommandProvider(characterId, worldId, targetId))
}
