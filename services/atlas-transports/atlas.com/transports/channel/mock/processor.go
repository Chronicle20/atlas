package mock

import (
	"atlas-transports/channel"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

// Compile-time interface compliance check
var _ channel.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the channel.Processor interface
type ProcessorMock struct {
	RegisterFunc   func(worldId world.Id, channelId channel2.Id) error
	UnregisterFunc func(worldId world.Id, channelId channel2.Id) error
	GetAllFunc     func() []channel2.Model
}

// Register is a mock implementation
func (m *ProcessorMock) Register(worldId world.Id, channelId channel2.Id) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(worldId, channelId)
	}
	return nil
}

// Unregister is a mock implementation
func (m *ProcessorMock) Unregister(worldId world.Id, channelId channel2.Id) error {
	if m.UnregisterFunc != nil {
		return m.UnregisterFunc(worldId, channelId)
	}
	return nil
}

// GetAll is a mock implementation
func (m *ProcessorMock) GetAll() []channel2.Model {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []channel2.Model{}
}
