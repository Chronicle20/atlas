package chair

import (
	chair2 "atlas-channel/kafka/message/chair"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for chair processing
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Use(f field.Model, chairType string, chairId uint32, characterId uint32) error
	Cancel(f field.Model, characterId uint32) error
	Recover(f field.Model, characterId uint32, hp int16, mp int16) error
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

var _ Processor = (*ProcessorImpl)(nil)

// InMapModelProvider fetches every occupied chair currently in one map
// instance (used to replay existing chair-sit state to a character entering
// the map). The upstream atlas-chairs list is now paginated (task-117), so
// this drains every page rather than fetching just the first.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) Use(f field.Model, chairType string, chairId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to use map [%d] [%s] chair [%d].", characterId, f.MapId(), chairType, chairId)
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(UseCommandProvider(f, chairType, chairId, characterId))
}

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(CancelCommandProvider(f, characterId))
}

func (p *ProcessorImpl) Recover(f field.Model, characterId uint32, hp int16, mp int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(RecoveryCommandProvider(f, characterId, hp, mp))
}
