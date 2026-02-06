package channel

import (
	tenant2 "atlas-world/configuration/tenant"
	"atlas-world/kafka/message"
	channel2 "atlas-world/kafka/message/channel"
	"atlas-world/kafka/producer"
	channel3 "atlas-world/kafka/producer/channel"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for channel processing operations
type Processor interface {
	AllProvider() model.Provider[[]Model]
	GetByWorld(worldId world.Id) ([]Model, error)
	ByWorldProvider(worldId world.Id) model.Provider[[]Model]
	GetById(ch channel.Model) (Model, error)
	ByIdProvider(ch channel.Model) model.Provider[Model]
	Register(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) (Model, error)
	Unregister(ch channel.Model) error
	RequestStatus(mb *message.Buffer) error
	RequestStatusAndEmit() error
	EmitStarted(mb *message.Buffer) func(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error
	EmitStartedAndEmit(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new channel processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// AllProvider returns all channel servers for the tenant
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.FixedProvider(GetChannelRegistry().ChannelServers(p.t))
}

func (p *ProcessorImpl) GetByWorld(worldId world.Id) ([]Model, error) {
	return p.ByWorldProvider(worldId)()
}

// ByWorldProvider returns all channel servers for a specific world
func (p *ProcessorImpl) ByWorldProvider(worldId world.Id) model.Provider[[]Model] {
	return model.FilteredProvider[Model](p.AllProvider(), model.Filters(ByWorldFilter(worldId)))
}

func (p *ProcessorImpl) GetById(ch channel.Model) (Model, error) {
	return p.ByIdProvider(ch)()
}

// ByIdProvider returns a specific channel server
func (p *ProcessorImpl) ByIdProvider(ch channel.Model) model.Provider[Model] {
	cs, err := GetChannelRegistry().ChannelServer(p.t, ch)
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(cs)
}

// Register registers a new channel server
func (p *ProcessorImpl) Register(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) (Model, error) {
	p.l.Debugf("Registering world [%d] channel [%d] for tenant [%s].", ch.WorldId(), ch.Id(), p.t.String())
	m, err := NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(ch.WorldId()).
		SetChannelId(ch.Id()).
		SetIpAddress(ipAddress).
		SetPort(port).
		SetCurrentCapacity(currentCapacity).
		SetMaxCapacity(maxCapacity).
		Build()
	if err != nil {
		return Model{}, err
	}
	return GetChannelRegistry().Register(p.t, m), nil
}

// Unregister unregisters a channel server
func (p *ProcessorImpl) Unregister(ch channel.Model) error {
	p.l.Debugf("Unregistering world [%d] channel [%d] for tenant [%s].", ch.WorldId(), ch.Id(), p.t.String())
	return GetChannelRegistry().RemoveByWorldAndChannel(p.t, ch)
}

// RequestStatus requests the status of channels for a tenant
func (p *ProcessorImpl) RequestStatus(mb *message.Buffer) error {
	p.l.Debugf("Requesting status of channels for tenant [%s].", p.t.String())
	err := mb.Put(channel2.EnvCommandTopic, channel3.StatusCommandProvider(p.t))
	if err != nil {
		return err
	}
	return nil
}

// RequestStatusAndEmit requests the status of channels for a tenant and emits the command
func (p *ProcessorImpl) RequestStatusAndEmit() error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		err := p.RequestStatus(buf)
		if err != nil {
			return err
		}
		return nil
	})
}

// EmitStarted returns a function that emits a channel started event to the message buffer
func (p *ProcessorImpl) EmitStarted(mb *message.Buffer) func(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
	return func(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
		return mb.Put(channel2.EnvEventTopicStatus, channel3.StartedEventProvider(p.t, ch, ipAddress, port, currentCapacity, maxCapacity))
	}
}

// EmitStartedAndEmit emits a channel started event
func (p *ProcessorImpl) EmitStartedAndEmit(ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.EmitStarted(buf)(ch, ipAddress, port, currentCapacity, maxCapacity)
	})
}

// ByWorldFilter creates a filter for channels by world ID
func ByWorldFilter(id world.Id) model.Filter[Model] {
	return func(m Model) bool {
		return m.WorldId() == id
	}
}

func RequestStatus(l logrus.FieldLogger) func(ctx context.Context) func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
	return func(ctx context.Context) func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
		return func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
			return func(rm tenant2.RestModel) error {
				t, err := tenant.Create(uuid.MustParse(rm.Id), rm.Region, rm.MajorVersion, rm.MinorVersion)
				if err != nil {
					return err
				}
				tctx := tenant.WithContext(ctx, t)
				return NewProcessor(l, tctx).RequestStatusAndEmit()
			}
		}
	}
}
