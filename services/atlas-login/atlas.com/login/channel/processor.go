package channel

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"math/rand"
)

type Processor interface {
	ByIdModelProvider(worldId byte, channelId byte) model.Provider[Model]
	GetById(worldId byte, channelId byte) (Model, error)
	ByWorldModelProvider(worldId byte) model.Provider[[]Model]
	GetForWorld(worldId byte) ([]Model, error)
	GetRandomInWorld(worldId byte) (Model, error)
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

func (p *ProcessorImpl) ByIdModelProvider(worldId byte, channelId byte) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(worldId, channelId), Extract)
}

func (p *ProcessorImpl) GetById(worldId byte, channelId byte) (Model, error) {
	return p.ByIdModelProvider(worldId, channelId)()
}

func (p *ProcessorImpl) ByWorldModelProvider(worldId byte) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestChannelsForWorld(worldId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetForWorld(worldId byte) ([]Model, error) {
	return p.ByWorldModelProvider(worldId)()
}

func (p *ProcessorImpl) GetRandomInWorld(worldId byte) (Model, error) {
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
