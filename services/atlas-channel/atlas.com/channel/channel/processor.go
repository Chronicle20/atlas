package channel

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for channel processing
type Processor interface {
	Register(worldId world.Id, channelId channel.Id, ipAddress string, port int) error
	ByIdModelProvider(worldId world.Id, channelId channel.Id) model.Provider[Model]
	GetById(worldId world.Id, channelId channel.Id) (Model, error)
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

func (p *ProcessorImpl) Register(worldId world.Id, channelId channel.Id, ipAddress string, port int) error {
	return registerChannel(p.l)(p.ctx)(NewBuilder().
		SetId(uuid.New()).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetIpAddress(ipAddress).
		SetPort(port).
		SetCurrentCapacity(0).
		SetMaxCapacity(1000).
		MustBuild())
}

func (p *ProcessorImpl) ByIdModelProvider(worldId world.Id, channelId channel.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(worldId, channelId), Extract)
}

func (p *ProcessorImpl) GetById(worldId world.Id, channelId channel.Id) (Model, error) {
	return p.ByIdModelProvider(worldId, channelId)()
}
