package buddylist

import (
	buddylist2 "atlas-channel/kafka/message/buddylist"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for buddylist processing
type Processor interface {
	GetById(characterId uint32) (Model, error)
	RequestAdd(characterId uint32, worldId world.Id, targetId uint32, group string) error
	RequestDelete(characterId uint32, worldId world.Id, targetId uint32) error
}

// ProcessorImpl implements the Processor interface
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

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()
}

func (p *ProcessorImpl) RequestAdd(characterId uint32, worldId world.Id, targetId uint32, group string) error {
	p.l.Debugf("Character [%d] would like to add [%d] to group [%s] to their buddy list.", characterId, targetId, group)
	return producer.ProviderImpl(p.l)(p.ctx)(buddylist2.EnvCommandTopic)(RequestAddBuddyCommandProvider(characterId, worldId, targetId, group))
}

func (p *ProcessorImpl) RequestDelete(characterId uint32, worldId world.Id, targetId uint32) error {
	p.l.Debugf("Character [%d] attempting to delete buddy [%d].", characterId, targetId)
	return producer.ProviderImpl(p.l)(p.ctx)(buddylist2.EnvCommandTopic)(RequestDeleteBuddyCommandProvider(characterId, worldId, targetId))
}
