package channel

import (
	"context"
	"errors"
	"math/rand"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *ProcessorImpl) ByIdModelProvider(ch channel.Model) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(ch), Extract)
}

func (p *ProcessorImpl) GetById(ch channel.Model) (Model, error) {
	return p.ByIdModelProvider(ch)()
}

func (p *ProcessorImpl) ByWorldModelProvider(worldId world.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestChannelsForWorld(worldId), Extract, model.Filters[Model]())
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
