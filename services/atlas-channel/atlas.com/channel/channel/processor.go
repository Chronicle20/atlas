package channel

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for channel processing
type Processor interface {
	Register(ch channel.Model, ipAddress string, port int) error
	Unregister(ch channel.Model) error
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

// Unregister DELETEs the (worldId, channelId) entry on atlas-world. A 404
// from upstream is treated as success — the listener drain may race with
// an operator-driven unregister, and the goal is "the channel is gone."
func (p *ProcessorImpl) Unregister(ch channel.Model) error {
	err := unregisterChannel(ch)(p.l, p.ctx)
	if err != nil && !errors.Is(err, requests.ErrNotFound) {
		return err
	}
	return nil
}

func (p *ProcessorImpl) ByIdModelProvider(ch channel.Model) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestChannel(ch), Extract)
}

func (p *ProcessorImpl) GetById(ch channel.Model) (Model, error) {
	return p.ByIdModelProvider(ch)()
}
