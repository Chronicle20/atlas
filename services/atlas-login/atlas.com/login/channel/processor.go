package channel

import (
	"context"
	"errors"
	"math/rand"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdModelProvider(ch channel.Model) model.Provider[Model]
	GetById(ch channel.Model) (Model, error)
	ByWorldModelProvider(worldId world.Id) model.Provider[[]Model]
	GetForWorld(worldId world.Id) ([]Model, error)
	GetRandomInWorld(worldId world.Id) (Model, error)
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

func (p *ProcessorImpl) ByIdModelProvider(ch channel.Model) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(ch), Extract)
}

func (p *ProcessorImpl) GetById(ch channel.Model) (Model, error) {
	return p.ByIdModelProvider(ch)()
}

// ByWorldModelProvider fetches every channel server registered for a world.
// The upstream atlas-world channels-for-world list is now paginated
// (task-117); this is a genuine startup consumer (GetRandomInWorld picks
// uniformly among every channel in the world when routing a logging-in
// character), so it drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByWorldModelProvider(worldId world.Id) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(channelsForWorldUrl(worldId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetForWorld(worldId world.Id) ([]Model, error) {
	return p.ByWorldModelProvider(worldId)()
}

func (p *ProcessorImpl) GetRandomInWorld(worldId world.Id) (Model, error) {
	cs, err := p.GetForWorld(worldId)
	if err != nil {
		return Model{}, err
	}
	if len(cs) == 0 {
		return Model{}, errors.New("no channel found")
	}

	ri := rand.Intn(len(cs))
	return cs[ri], nil
}
