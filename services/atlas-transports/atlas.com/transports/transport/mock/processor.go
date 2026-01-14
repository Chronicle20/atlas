package mock

import (
	"atlas-transports/kafka/message"
	"atlas-transports/transport"

	"github.com/Chronicle20/atlas-constants/field"
	map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// Compile-time interface compliance check
var _ transport.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the transport.Processor interface
type ProcessorMock struct {
	AddTenantFunc                          func(routes []transport.Model, sharedVessels []transport.SharedVesselModel) error
	ClearTenantFunc                        func() int
	ByIdProviderFunc                       func(id uuid.UUID) model.Provider[transport.Model]
	ByStartMapProviderFunc                 func(mapId map2.Id) model.Provider[transport.Model]
	GetByStartMapFunc                      func(mapId map2.Id) (transport.Model, error)
	AllRoutesProviderFunc                  func() model.Provider[[]transport.Model]
	UpdateRoutesFunc                       func() error
	UpdateRouteAndEmitFunc                 func(route transport.Model) error
	WarpToRouteStartMapOnLogoutFunc        func(mb *message.Buffer) func(characterId uint32, f field.Model) error
	WarpToRouteStartMapOnLogoutAndEmitFunc func(characterId uint32, f field.Model) error
}

// AddTenant is a mock implementation
func (m *ProcessorMock) AddTenant(routes []transport.Model, sharedVessels []transport.SharedVesselModel) error {
	if m.AddTenantFunc != nil {
		return m.AddTenantFunc(routes, sharedVessels)
	}
	return nil
}

// ClearTenant is a mock implementation
func (m *ProcessorMock) ClearTenant() int {
	if m.ClearTenantFunc != nil {
		return m.ClearTenantFunc()
	}
	return 0
}

// ByIdProvider is a mock implementation
func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[transport.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (transport.Model, error) {
		return transport.Model{}, nil
	}
}

// ByStartMapProvider is a mock implementation
func (m *ProcessorMock) ByStartMapProvider(mapId map2.Id) model.Provider[transport.Model] {
	if m.ByStartMapProviderFunc != nil {
		return m.ByStartMapProviderFunc(mapId)
	}
	return func() (transport.Model, error) {
		return transport.Model{}, nil
	}
}

// GetByStartMap is a mock implementation
func (m *ProcessorMock) GetByStartMap(mapId map2.Id) (transport.Model, error) {
	if m.GetByStartMapFunc != nil {
		return m.GetByStartMapFunc(mapId)
	}
	return transport.Model{}, nil
}

// AllRoutesProvider is a mock implementation
func (m *ProcessorMock) AllRoutesProvider() model.Provider[[]transport.Model] {
	if m.AllRoutesProviderFunc != nil {
		return m.AllRoutesProviderFunc()
	}
	return func() ([]transport.Model, error) {
		return []transport.Model{}, nil
	}
}

// UpdateRoutes is a mock implementation
func (m *ProcessorMock) UpdateRoutes() error {
	if m.UpdateRoutesFunc != nil {
		return m.UpdateRoutesFunc()
	}
	return nil
}

// UpdateRouteAndEmit is a mock implementation
func (m *ProcessorMock) UpdateRouteAndEmit(route transport.Model) error {
	if m.UpdateRouteAndEmitFunc != nil {
		return m.UpdateRouteAndEmitFunc(route)
	}
	return nil
}

// WarpToRouteStartMapOnLogout is a mock implementation
func (m *ProcessorMock) WarpToRouteStartMapOnLogout(mb *message.Buffer) func(characterId uint32, f field.Model) error {
	if m.WarpToRouteStartMapOnLogoutFunc != nil {
		return m.WarpToRouteStartMapOnLogoutFunc(mb)
	}
	return func(characterId uint32, f field.Model) error {
		return nil
	}
}

// WarpToRouteStartMapOnLogoutAndEmit is a mock implementation
func (m *ProcessorMock) WarpToRouteStartMapOnLogoutAndEmit(characterId uint32, f field.Model) error {
	if m.WarpToRouteStartMapOnLogoutAndEmitFunc != nil {
		return m.WarpToRouteStartMapOnLogoutAndEmitFunc(characterId, f)
	}
	return nil
}
