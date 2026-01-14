package mock

import (
	"atlas-login/channel"

	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a mock implementation of channel.Processor for testing
type MockProcessor struct {
	ByIdModelProviderFunc    func(worldId byte, channelId byte) model.Provider[channel.Model]
	GetByIdFunc              func(worldId byte, channelId byte) (channel.Model, error)
	ByWorldModelProviderFunc func(worldId byte) model.Provider[[]channel.Model]
	GetForWorldFunc          func(worldId byte) ([]channel.Model, error)
	GetRandomInWorldFunc     func(worldId byte) (channel.Model, error)
}

// ByIdModelProvider implements channel.Processor
func (m *MockProcessor) ByIdModelProvider(worldId byte, channelId byte) model.Provider[channel.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(worldId, channelId)
	}
	return func() (channel.Model, error) {
		return channel.Model{}, nil
	}
}

// GetById implements channel.Processor
func (m *MockProcessor) GetById(worldId byte, channelId byte) (channel.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(worldId, channelId)
	}
	return channel.Model{}, nil
}

// ByWorldModelProvider implements channel.Processor
func (m *MockProcessor) ByWorldModelProvider(worldId byte) model.Provider[[]channel.Model] {
	if m.ByWorldModelProviderFunc != nil {
		return m.ByWorldModelProviderFunc(worldId)
	}
	return func() ([]channel.Model, error) {
		return []channel.Model{}, nil
	}
}

// GetForWorld implements channel.Processor
func (m *MockProcessor) GetForWorld(worldId byte) ([]channel.Model, error) {
	if m.GetForWorldFunc != nil {
		return m.GetForWorldFunc(worldId)
	}
	return []channel.Model{}, nil
}

// GetRandomInWorld implements channel.Processor
func (m *MockProcessor) GetRandomInWorld(worldId byte) (channel.Model, error) {
	if m.GetRandomInWorldFunc != nil {
		return m.GetRandomInWorldFunc(worldId)
	}
	return channel.Model{}, nil
}

// Verify MockProcessor implements channel.Processor
var _ channel.Processor = (*MockProcessor)(nil)
