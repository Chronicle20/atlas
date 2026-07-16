package reactor

import (
	reactor2 "atlas-channel/kafka/message/reactor"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Hit(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) error
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

// InMapModelProvider fetches every reactor currently in one map instance.
// This is a hot-path consumer (reactor spawn/state on every channel spawn
// broadcast, ForEachInMap for hit-detection); the upstream atlas-reactors
// list is now paginated (task-117), so this drains every page rather than
// fetching just the first -- a truncated list here means reactors silently
// vanish from the client's view.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) Hit(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) error {
	p.l.Debugf("Sending hit command for reactor [%d]. CharacterId [%d]. Stance [%d]. SkillId [%d].", reactorId, characterId, stance, skillId)
	return producer.ProviderImpl(p.l)(p.ctx)(reactor2.EnvCommandTopic)(HitCommandProvider(f, reactorId, characterId, stance, skillId))
}
