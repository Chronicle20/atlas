package mock

import (
	"atlas-world/channel"
	"atlas-world/kafka/message"

	channelConstant "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

// Processor is a mock implementation of channel.Processor for testing
type Processor struct {
	AllProviderFunc          func() model.Provider[[]channel.Model]
	GetByWorldFunc           func(worldId world.Id) ([]channel.Model, error)
	ByWorldProviderFunc      func(worldId world.Id) model.Provider[[]channel.Model]
	GetByIdFunc              func(worldId world.Id, channelId channelConstant.Id) (channel.Model, error)
	ByIdProviderFunc         func(worldId world.Id, channelId channelConstant.Id) model.Provider[channel.Model]
	RegisterFunc             func(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) (channel.Model, error)
	UnregisterFunc           func(worldId world.Id, channelId channelConstant.Id) error
	RequestStatusFunc        func(mb *message.Buffer) error
	RequestStatusAndEmitFunc func() error
	EmitStartedFunc          func(mb *message.Buffer) func(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error
	EmitStartedAndEmitFunc   func(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error
}

// Compile-time interface check
var _ channel.Processor = (*Processor)(nil)

func (m *Processor) AllProvider() model.Provider[[]channel.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider[[]channel.Model](nil)
}

func (m *Processor) GetByWorld(worldId world.Id) ([]channel.Model, error) {
	if m.GetByWorldFunc != nil {
		return m.GetByWorldFunc(worldId)
	}
	return nil, nil
}

func (m *Processor) ByWorldProvider(worldId world.Id) model.Provider[[]channel.Model] {
	if m.ByWorldProviderFunc != nil {
		return m.ByWorldProviderFunc(worldId)
	}
	return model.FixedProvider[[]channel.Model](nil)
}

func (m *Processor) GetById(worldId world.Id, channelId channelConstant.Id) (channel.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(worldId, channelId)
	}
	return channel.Model{}, nil
}

func (m *Processor) ByIdProvider(worldId world.Id, channelId channelConstant.Id) model.Provider[channel.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(worldId, channelId)
	}
	return model.FixedProvider[channel.Model](channel.Model{})
}

func (m *Processor) Register(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) (channel.Model, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(worldId, channelId, ipAddress, port, currentCapacity, maxCapacity)
	}
	return channel.Model{}, nil
}

func (m *Processor) Unregister(worldId world.Id, channelId channelConstant.Id) error {
	if m.UnregisterFunc != nil {
		return m.UnregisterFunc(worldId, channelId)
	}
	return nil
}

func (m *Processor) RequestStatus(mb *message.Buffer) error {
	if m.RequestStatusFunc != nil {
		return m.RequestStatusFunc(mb)
	}
	return nil
}

func (m *Processor) RequestStatusAndEmit() error {
	if m.RequestStatusAndEmitFunc != nil {
		return m.RequestStatusAndEmitFunc()
	}
	return nil
}

func (m *Processor) EmitStarted(mb *message.Buffer) func(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
	if m.EmitStartedFunc != nil {
		return m.EmitStartedFunc(mb)
	}
	return func(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
		return nil
	}
}

func (m *Processor) EmitStartedAndEmit(worldId world.Id, channelId channelConstant.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) error {
	if m.EmitStartedAndEmitFunc != nil {
		return m.EmitStartedAndEmitFunc(worldId, channelId, ipAddress, port, currentCapacity, maxCapacity)
	}
	return nil
}
