package mock

import (
	"atlas-login/channel"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a mock implementation of channel.Processor for testing
type MockProcessor struct {
	ByIdModelProviderFunc    func(ch channel2.Model) model.Provider[channel.Model]
	GetByIdFunc              func(ch channel2.Model) (channel.Model, error)
	ByWorldModelProviderFunc func(worldId world.Id) model.Provider[[]channel.Model]
	GetForWorldFunc          func(worldId world.Id) ([]channel.Model, error)
	GetRandomInWorldFunc     func(worldId world.Id) (channel.Model, error)
}

// ByIdModelProvider implements channel.Processor
func (m *MockProcessor) ByIdModelProvider(ch channel2.Model) model.Provider[channel.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(ch)
	}
	return func() (channel.Model, error) {
		return channel.Model{}, nil
	}
}

// GetById implements channel.Processor
func (m *MockProcessor) GetById(ch channel2.Model) (channel.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(ch)
	}
	return channel.Model{}, nil
}

// ByWorldModelProvider implements channel.Processor
func (m *MockProcessor) ByWorldModelProvider(worldId world.Id) model.Provider[[]channel.Model] {
	if m.ByWorldModelProviderFunc != nil {
		return m.ByWorldModelProviderFunc(worldId)
	}
	return func() ([]channel.Model, error) {
		return []channel.Model{}, nil
	}
}

// GetForWorld implements channel.Processor
func (m *MockProcessor) GetForWorld(worldId world.Id) ([]channel.Model, error) {
	if m.GetForWorldFunc != nil {
		return m.GetForWorldFunc(worldId)
	}
	return []channel.Model{}, nil
}

// GetRandomInWorld implements channel.Processor
func (m *MockProcessor) GetRandomInWorld(worldId world.Id) (channel.Model, error) {
	if m.GetRandomInWorldFunc != nil {
		return m.GetRandomInWorldFunc(worldId)
	}
	return channel.Model{}, nil
}

// Verify MockProcessor implements channel.Processor
var _ channel.Processor = (*MockProcessor)(nil)
