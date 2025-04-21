package world

import (
	"atlas-login/channel"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  *channel.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		cp:  channel.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestWorlds(), Extract, model.Filters[Model]())
}

func (p *Processor) GetAll(decorators ...model.Decorator[Model]) ([]Model, error) {
	return model.SliceMap(model.Decorate(decorators))(p.AllProvider())(model.ParallelMap())()
}

func (p *Processor) ByIdModelProvider(worldId byte) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestWorld(worldId), Extract)
}

func (p *Processor) GetById(worldId byte) (Model, error) {
	return p.ByIdModelProvider(worldId)()
}

func (p *Processor) GetCapacityStatus(worldId byte) Status {
	w, err := p.GetById(worldId)
	if err != nil {
		return StatusFull
	}
	return w.CapacityStatus()
}

func (p *Processor) ChannelLoadDecorator(m Model) Model {
	nm, err := model.Fold[channel.Model, Model](p.cp.ByWorldModelProvider(m.Id()), Clone(m), foldChannelLoad)()
	if err != nil {
		return m
	}
	return nm
}

func foldChannelLoad(m Model, c channel.Model) (Model, error) {
	return CloneWorld(m).AddChannelLoad(c.ChannelId(), c.Capacity()).Build(), nil
}
