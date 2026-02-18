package channel

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Register(ch channel.Model) error
	Unregister(ch channel.Model) error
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

func (p *ProcessorImpl) Register(ch channel.Model) error {
	getRegistry().Add(p.ctx, ch)
	return nil
}

func (p *ProcessorImpl) Unregister(ch channel.Model) error {
	getRegistry().Remove(p.ctx, ch)
	return nil
}

func (p *ProcessorImpl) GetAll() []channel.Model {
	return getRegistry().GetAll(p.ctx)
}
