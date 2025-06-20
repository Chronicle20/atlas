package channel

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) Register(worldId world.Id, channelId channel.Id, ipAddress string, port int) error {
	return registerChannel(p.l)(p.ctx)(NewBuilder().
		SetWorldId(byte(worldId)).
		SetChannelId(byte(channelId)).
		SetIpAddress(ipAddress).
		SetPort(port).
		SetCurrentCapacity(0).
		SetMaxCapacity(0).
		Build())
}

func (p *Processor) ByIdModelProvider(worldId world.Id, channelId channel.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(worldId, channelId), Extract)
}

func (p *Processor) GetById(worldId world.Id, channelId channel.Id) (Model, error) {
	return p.ByIdModelProvider(worldId, channelId)()
}
