package mock

import (
	"atlas-transports/instance"
	"atlas-transports/kafka/message"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

var _ instance.Processor = (*ProcessorMock)(nil)

type ProcessorMock struct {
	AddTenantFunc                    func(routes []instance.RouteModel)
	ClearTenantFunc                  func() int
	GetRoutesFunc                    func() []instance.RouteModel
	GetRouteFunc                     func(id uuid.UUID) (instance.RouteModel, bool)
	IsTransitMapFunc                 func(mapId _map.Id) bool
	GetRouteByTransitMapFunc         func(mapId _map.Id) (instance.RouteModel, error)
	StartTransportFunc               func(mb *message.Buffer) func(characterId uint32, routeId uuid.UUID, f field.Model) error
	StartTransportAndEmitFunc        func(characterId uint32, routeId uuid.UUID, f field.Model) error
	HandleMapEnterFunc               func(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleMapEnterAndEmitFunc        func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleMapExitFunc                func(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleMapExitAndEmitFunc         func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleLogoutFunc                 func(mb *message.Buffer) func(characterId uint32, worldId world.Id, channelId channel.Id) error
	HandleLogoutAndEmitFunc          func(characterId uint32, worldId world.Id, channelId channel.Id) error
	HandleLoginFunc                  func(mb *message.Buffer) func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error
	HandleLoginAndEmitFunc           func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error
	TickBoardingExpirationFunc       func(mb *message.Buffer) error
	TickBoardingExpirationAndEmitFunc func() error
	TickArrivalFunc                  func(mb *message.Buffer) error
	TickArrivalAndEmitFunc           func() error
	TickStuckTimeoutFunc             func(mb *message.Buffer) error
	TickStuckTimeoutAndEmitFunc      func() error
	GracefulShutdownFunc             func(mb *message.Buffer) error
	GracefulShutdownAndEmitFunc      func() error
}

func (m *ProcessorMock) AddTenant(routes []instance.RouteModel) {
	if m.AddTenantFunc != nil {
		m.AddTenantFunc(routes)
	}
}

func (m *ProcessorMock) ClearTenant() int {
	if m.ClearTenantFunc != nil {
		return m.ClearTenantFunc()
	}
	return 0
}

func (m *ProcessorMock) GetRoutes() []instance.RouteModel {
	if m.GetRoutesFunc != nil {
		return m.GetRoutesFunc()
	}
	return []instance.RouteModel{}
}

func (m *ProcessorMock) GetRoute(id uuid.UUID) (instance.RouteModel, bool) {
	if m.GetRouteFunc != nil {
		return m.GetRouteFunc(id)
	}
	return instance.RouteModel{}, false
}

func (m *ProcessorMock) IsTransitMap(mapId _map.Id) bool {
	if m.IsTransitMapFunc != nil {
		return m.IsTransitMapFunc(mapId)
	}
	return false
}

func (m *ProcessorMock) GetRouteByTransitMap(mapId _map.Id) (instance.RouteModel, error) {
	if m.GetRouteByTransitMapFunc != nil {
		return m.GetRouteByTransitMapFunc(mapId)
	}
	return instance.RouteModel{}, nil
}

func (m *ProcessorMock) StartTransport(mb *message.Buffer) func(characterId uint32, routeId uuid.UUID, f field.Model) error {
	if m.StartTransportFunc != nil {
		return m.StartTransportFunc(mb)
	}
	return func(characterId uint32, routeId uuid.UUID, f field.Model) error {
		return nil
	}
}

func (m *ProcessorMock) StartTransportAndEmit(characterId uint32, routeId uuid.UUID, f field.Model) error {
	if m.StartTransportAndEmitFunc != nil {
		return m.StartTransportAndEmitFunc(characterId, routeId, f)
	}
	return nil
}

func (m *ProcessorMock) HandleMapEnter(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	if m.HandleMapEnterFunc != nil {
		return m.HandleMapEnterFunc(mb)
	}
	return func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
		return nil
	}
}

func (m *ProcessorMock) HandleMapEnterAndEmit(characterId uint32, mapId _map.Id, inst uuid.UUID, worldId world.Id, channelId channel.Id) error {
	if m.HandleMapEnterAndEmitFunc != nil {
		return m.HandleMapEnterAndEmitFunc(characterId, mapId, inst, worldId, channelId)
	}
	return nil
}

func (m *ProcessorMock) HandleMapExit(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	if m.HandleMapExitFunc != nil {
		return m.HandleMapExitFunc(mb)
	}
	return func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
		return nil
	}
}

func (m *ProcessorMock) HandleMapExitAndEmit(characterId uint32, mapId _map.Id, inst uuid.UUID, worldId world.Id, channelId channel.Id) error {
	if m.HandleMapExitAndEmitFunc != nil {
		return m.HandleMapExitAndEmitFunc(characterId, mapId, inst, worldId, channelId)
	}
	return nil
}

func (m *ProcessorMock) HandleLogout(mb *message.Buffer) func(characterId uint32, worldId world.Id, channelId channel.Id) error {
	if m.HandleLogoutFunc != nil {
		return m.HandleLogoutFunc(mb)
	}
	return func(characterId uint32, worldId world.Id, channelId channel.Id) error {
		return nil
	}
}

func (m *ProcessorMock) HandleLogoutAndEmit(characterId uint32, worldId world.Id, channelId channel.Id) error {
	if m.HandleLogoutAndEmitFunc != nil {
		return m.HandleLogoutAndEmitFunc(characterId, worldId, channelId)
	}
	return nil
}

func (m *ProcessorMock) HandleLogin(mb *message.Buffer) func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
	if m.HandleLoginFunc != nil {
		return m.HandleLoginFunc(mb)
	}
	return func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
		return nil
	}
}

func (m *ProcessorMock) HandleLoginAndEmit(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
	if m.HandleLoginAndEmitFunc != nil {
		return m.HandleLoginAndEmitFunc(characterId, mapId, worldId, channelId)
	}
	return nil
}

func (m *ProcessorMock) TickBoardingExpiration(mb *message.Buffer) error {
	if m.TickBoardingExpirationFunc != nil {
		return m.TickBoardingExpirationFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickBoardingExpirationAndEmit() error {
	if m.TickBoardingExpirationAndEmitFunc != nil {
		return m.TickBoardingExpirationAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) TickArrival(mb *message.Buffer) error {
	if m.TickArrivalFunc != nil {
		return m.TickArrivalFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickArrivalAndEmit() error {
	if m.TickArrivalAndEmitFunc != nil {
		return m.TickArrivalAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) TickStuckTimeout(mb *message.Buffer) error {
	if m.TickStuckTimeoutFunc != nil {
		return m.TickStuckTimeoutFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickStuckTimeoutAndEmit() error {
	if m.TickStuckTimeoutAndEmitFunc != nil {
		return m.TickStuckTimeoutAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) GracefulShutdown(mb *message.Buffer) error {
	if m.GracefulShutdownFunc != nil {
		return m.GracefulShutdownFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) GracefulShutdownAndEmit() error {
	if m.GracefulShutdownAndEmitFunc != nil {
		return m.GracefulShutdownAndEmitFunc()
	}
	return nil
}
