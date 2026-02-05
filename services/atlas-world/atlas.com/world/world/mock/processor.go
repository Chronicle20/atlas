package mock

import (
	"atlas-world/world"

	worldConstant "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

// Processor is a mock implementation of world.Processor for testing
type Processor struct {
	ChannelDecoratorFunc  func(m world.Model) world.Model
	GetWorldsFunc         func(decorators ...model.Decorator[world.Model]) ([]world.Model, error)
	AllWorldProviderFunc  func(decorators ...model.Decorator[world.Model]) model.Provider[[]world.Model]
	GetWorldFunc          func(decorators ...model.Decorator[world.Model]) func(worldId worldConstant.Id) (world.Model, error)
	ByWorldIdProviderFunc func(decorators ...model.Decorator[world.Model]) func(worldId worldConstant.Id) model.Provider[world.Model]
}

// Compile-time interface check
var _ world.Processor = (*Processor)(nil)

func (m *Processor) ChannelDecorator(model world.Model) world.Model {
	if m.ChannelDecoratorFunc != nil {
		return m.ChannelDecoratorFunc(model)
	}
	return model
}

func (m *Processor) GetWorlds(decorators ...model.Decorator[world.Model]) ([]world.Model, error) {
	if m.GetWorldsFunc != nil {
		return m.GetWorldsFunc(decorators...)
	}
	return nil, nil
}

func (m *Processor) AllWorldProvider(decorators ...model.Decorator[world.Model]) model.Provider[[]world.Model] {
	if m.AllWorldProviderFunc != nil {
		return m.AllWorldProviderFunc(decorators...)
	}
	return model.FixedProvider[[]world.Model](nil)
}

func (m *Processor) GetWorld(decorators ...model.Decorator[world.Model]) func(worldId worldConstant.Id) (world.Model, error) {
	if m.GetWorldFunc != nil {
		return m.GetWorldFunc(decorators...)
	}
	return func(worldId worldConstant.Id) (world.Model, error) {
		return world.Model{}, nil
	}
}

func (m *Processor) ByWorldIdProvider(decorators ...model.Decorator[world.Model]) func(worldId worldConstant.Id) model.Provider[world.Model] {
	if m.ByWorldIdProviderFunc != nil {
		return m.ByWorldIdProviderFunc(decorators...)
	}
	return func(worldId worldConstant.Id) model.Provider[world.Model] {
		return model.FixedProvider[world.Model](world.Model{})
	}
}
