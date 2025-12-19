package channel

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Register(worldId world.Id, channelId channel.Id) error
	Unregister(worldId world.Id, channelId channel.Id) error
	GetAll() []channel.Model
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) Register(worldId world.Id, channelId channel.Id) error {
	getRegistry().Add(p.t.Id(), channel.NewModel(worldId, channelId))
	return nil
}

func (p *ProcessorImpl) Unregister(worldId world.Id, channelId channel.Id) error {
	getRegistry().Remove(p.t.Id(), worldId, channelId)
	return nil
}

func (p *ProcessorImpl) GetAll() []channel.Model {
	return getRegistry().GetAll(p.t.Id())
}
