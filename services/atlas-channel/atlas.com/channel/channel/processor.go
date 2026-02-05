package channel

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for channel processing
type Processor interface {
	Register(ch channel.Model, ipAddress string, port int) error
	ByIdModelProvider(ch channel.Model) model.Provider[Model]
	GetById(ch channel.Model) (Model, error)
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

func (p *ProcessorImpl) Register(ch channel.Model, ipAddress string, port int) error {
	return registerChannel(p.l)(p.ctx)(NewBuilder().
		SetId(uuid.New()).
		SetWorldId(ch.WorldId()).
		SetChannelId(ch.Id()).
		SetIpAddress(ipAddress).
		SetPort(port).
		SetCurrentCapacity(0).
		SetMaxCapacity(1000).
		MustBuild())
}

func (p *ProcessorImpl) ByIdModelProvider(ch channel.Model) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(ch), Extract)
}

func (p *ProcessorImpl) GetById(ch channel.Model) (Model, error) {
	return p.ByIdModelProvider(ch)()
}
