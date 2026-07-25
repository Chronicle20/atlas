package mock

import (
	"atlas-tenants/configuration"
	"atlas-tenants/kafka/message"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Compile-time interface compliance check
var _ configuration.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the configuration.Processor interface
type ProcessorMock struct {
	// Route operations
	CreateRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error)
	CreateRouteAndEmitFunc func(tenantID uuid.UUID, route map[string]interface{}) (configuration.Model, error)
	UpdateRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error)
	UpdateRouteAndEmitFunc func(tenantID uuid.UUID, routeID string, route map[string]interface{}) (configuration.Model, error)
	DeleteRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error
	DeleteRouteAndEmitFunc func(tenantID uuid.UUID, routeID string) error
	GetRouteByIdFunc       func(tenantID uuid.UUID, routeID string) (map[string]interface{}, error)
	GetAllRoutesFunc       func(tenantID uuid.UUID) ([]map[string]interface{}, error)
	RouteByIdProviderFunc  func(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	AllRoutesProviderFunc  func(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// Instance route operations
	CreateInstanceRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error)
	CreateInstanceRouteAndEmitFunc func(tenantID uuid.UUID, route map[string]interface{}) (configuration.Model, error)
	UpdateInstanceRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error)
	UpdateInstanceRouteAndEmitFunc func(tenantID uuid.UUID, routeID string, route map[string]interface{}) (configuration.Model, error)
	DeleteInstanceRouteFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error
	DeleteInstanceRouteAndEmitFunc func(tenantID uuid.UUID, routeID string) error
	GetInstanceRouteByIdFunc       func(tenantID uuid.UUID, routeID string) (map[string]interface{}, error)
	GetAllInstanceRoutesFunc       func(tenantID uuid.UUID) ([]map[string]interface{}, error)
	InstanceRouteByIdProviderFunc  func(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	AllInstanceRoutesProviderFunc  func(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// MTS config operations
	CreateMtsConfigFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(config map[string]interface{}) (configuration.Model, error)
	CreateMtsConfigAndEmitFunc func(tenantID uuid.UUID, config map[string]interface{}) (configuration.Model, error)
	UpdateMtsConfigFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(configID string) func(config map[string]interface{}) (configuration.Model, error)
	UpdateMtsConfigAndEmitFunc func(tenantID uuid.UUID, configID string, config map[string]interface{}) (configuration.Model, error)
	DeleteMtsConfigFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(configID string) error
	DeleteMtsConfigAndEmitFunc func(tenantID uuid.UUID, configID string) error
	GetMtsConfigByIdFunc       func(tenantID uuid.UUID, configID string) (map[string]interface{}, error)
	GetAllMtsConfigsFunc       func(tenantID uuid.UUID) ([]map[string]interface{}, error)
	MtsConfigByIdProviderFunc  func(tenantID uuid.UUID, configID string) model.Provider[map[string]interface{}]
	AllMtsConfigsProviderFunc  func(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// Seed operations
	SeedRoutesFunc         func(tenantID uuid.UUID) (configuration.SeedResult, error)
	SeedInstanceRoutesFunc func(tenantID uuid.UUID) (configuration.SeedResult, error)
	SeedVesselsFunc        func(tenantID uuid.UUID) (configuration.SeedResult, error)
	SeedRpsRewardsFunc     func(tenantID uuid.UUID) (configuration.SeedResult, error)
	SeedMtsConfigsFunc     func(tenantID uuid.UUID) (configuration.SeedResult, error)

	// Vessel operations
	CreateVesselFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(vessel map[string]interface{}) (configuration.Model, error)
	CreateVesselAndEmitFunc func(tenantID uuid.UUID, vessel map[string]interface{}) (configuration.Model, error)
	UpdateVesselFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (configuration.Model, error)
	UpdateVesselAndEmitFunc func(tenantID uuid.UUID, vesselID string, vessel map[string]interface{}) (configuration.Model, error)
	DeleteVesselFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) error
	DeleteVesselAndEmitFunc func(tenantID uuid.UUID, vesselID string) error
	GetVesselByIdFunc       func(tenantID uuid.UUID, vesselID string) (map[string]interface{}, error)
	GetAllVesselsFunc       func(tenantID uuid.UUID) ([]map[string]interface{}, error)
	VesselByIdProviderFunc  func(tenantID uuid.UUID, vesselID string) model.Provider[map[string]interface{}]
	AllVesselsProviderFunc  func(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// RPS reward operations
	CreateRpsRewardFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsReward map[string]interface{}) (configuration.Model, error)
	CreateRpsRewardAndEmitFunc func(tenantID uuid.UUID, rpsReward map[string]interface{}) (configuration.Model, error)
	UpdateRpsRewardFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (configuration.Model, error)
	UpdateRpsRewardAndEmitFunc func(tenantID uuid.UUID, rpsRewardID string, rpsReward map[string]interface{}) (configuration.Model, error)
	DeleteRpsRewardFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsRewardID string) error
	DeleteRpsRewardAndEmitFunc func(tenantID uuid.UUID, rpsRewardID string) error
	GetRpsRewardByIdFunc       func(tenantID uuid.UUID, rpsRewardID string) (map[string]interface{}, error)
	GetAllRpsRewardsFunc       func(tenantID uuid.UUID) ([]map[string]interface{}, error)
	RpsRewardByIdProviderFunc  func(tenantID uuid.UUID, rpsRewardID string) model.Provider[map[string]interface{}]
	AllRpsRewardsProviderFunc  func(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]
}

// CreateRoute is a mock implementation
func (m *ProcessorMock) CreateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error) {
	if m.CreateRouteFunc != nil {
		return m.CreateRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error) {
		return func(route map[string]interface{}) (configuration.Model, error) {
			return configuration.Model{}, nil
		}
	}
}

// CreateRouteAndEmit is a mock implementation
func (m *ProcessorMock) CreateRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (configuration.Model, error) {
	if m.CreateRouteAndEmitFunc != nil {
		return m.CreateRouteAndEmitFunc(tenantID, route)
	}
	return configuration.Model{}, nil
}

// UpdateRoute is a mock implementation
func (m *ProcessorMock) UpdateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
	if m.UpdateRouteFunc != nil {
		return m.UpdateRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
		return func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
			return func(route map[string]interface{}) (configuration.Model, error) {
				return configuration.Model{}, nil
			}
		}
	}
}

// UpdateRouteAndEmit is a mock implementation
func (m *ProcessorMock) UpdateRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (configuration.Model, error) {
	if m.UpdateRouteAndEmitFunc != nil {
		return m.UpdateRouteAndEmitFunc(tenantID, routeID, route)
	}
	return configuration.Model{}, nil
}

// DeleteRoute is a mock implementation
func (m *ProcessorMock) DeleteRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	if m.DeleteRouteFunc != nil {
		return m.DeleteRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			return nil
		}
	}
}

// DeleteRouteAndEmit is a mock implementation
func (m *ProcessorMock) DeleteRouteAndEmit(tenantID uuid.UUID, routeID string) error {
	if m.DeleteRouteAndEmitFunc != nil {
		return m.DeleteRouteAndEmitFunc(tenantID, routeID)
	}
	return nil
}

// GetRouteById is a mock implementation
func (m *ProcessorMock) GetRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error) {
	if m.GetRouteByIdFunc != nil {
		return m.GetRouteByIdFunc(tenantID, routeID)
	}
	return map[string]interface{}{}, nil
}

// GetAllRoutes is a mock implementation
func (m *ProcessorMock) GetAllRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if m.GetAllRoutesFunc != nil {
		return m.GetAllRoutesFunc(tenantID)
	}
	return []map[string]interface{}{}, nil
}

// RouteByIdProvider is a mock implementation
func (m *ProcessorMock) RouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	if m.RouteByIdProviderFunc != nil {
		return m.RouteByIdProviderFunc(tenantID, routeID)
	}
	return func() (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
}

// AllRoutesProvider is a mock implementation
func (m *ProcessorMock) AllRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	if m.AllRoutesProviderFunc != nil {
		return m.AllRoutesProviderFunc(tenantID)
	}
	return func() ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
}

// CreateVessel is a mock implementation
func (m *ProcessorMock) CreateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vessel map[string]interface{}) (configuration.Model, error) {
	if m.CreateVesselFunc != nil {
		return m.CreateVesselFunc(mb)
	}
	return func(tenantID uuid.UUID) func(vessel map[string]interface{}) (configuration.Model, error) {
		return func(vessel map[string]interface{}) (configuration.Model, error) {
			return configuration.Model{}, nil
		}
	}
}

// CreateVesselAndEmit is a mock implementation
func (m *ProcessorMock) CreateVesselAndEmit(tenantID uuid.UUID, vessel map[string]interface{}) (configuration.Model, error) {
	if m.CreateVesselAndEmitFunc != nil {
		return m.CreateVesselAndEmitFunc(tenantID, vessel)
	}
	return configuration.Model{}, nil
}

// UpdateVessel is a mock implementation
func (m *ProcessorMock) UpdateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (configuration.Model, error) {
	if m.UpdateVesselFunc != nil {
		return m.UpdateVesselFunc(mb)
	}
	return func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (configuration.Model, error) {
		return func(vesselID string) func(vessel map[string]interface{}) (configuration.Model, error) {
			return func(vessel map[string]interface{}) (configuration.Model, error) {
				return configuration.Model{}, nil
			}
		}
	}
}

// UpdateVesselAndEmit is a mock implementation
func (m *ProcessorMock) UpdateVesselAndEmit(tenantID uuid.UUID, vesselID string, vessel map[string]interface{}) (configuration.Model, error) {
	if m.UpdateVesselAndEmitFunc != nil {
		return m.UpdateVesselAndEmitFunc(tenantID, vesselID, vessel)
	}
	return configuration.Model{}, nil
}

// DeleteVessel is a mock implementation
func (m *ProcessorMock) DeleteVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) error {
	if m.DeleteVesselFunc != nil {
		return m.DeleteVesselFunc(mb)
	}
	return func(tenantID uuid.UUID) func(vesselID string) error {
		return func(vesselID string) error {
			return nil
		}
	}
}

// DeleteVesselAndEmit is a mock implementation
func (m *ProcessorMock) DeleteVesselAndEmit(tenantID uuid.UUID, vesselID string) error {
	if m.DeleteVesselAndEmitFunc != nil {
		return m.DeleteVesselAndEmitFunc(tenantID, vesselID)
	}
	return nil
}

// GetVesselById is a mock implementation
func (m *ProcessorMock) GetVesselById(tenantID uuid.UUID, vesselID string) (map[string]interface{}, error) {
	if m.GetVesselByIdFunc != nil {
		return m.GetVesselByIdFunc(tenantID, vesselID)
	}
	return map[string]interface{}{}, nil
}

// GetAllVessels is a mock implementation
func (m *ProcessorMock) GetAllVessels(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if m.GetAllVesselsFunc != nil {
		return m.GetAllVesselsFunc(tenantID)
	}
	return []map[string]interface{}{}, nil
}

// VesselByIdProvider is a mock implementation
func (m *ProcessorMock) VesselByIdProvider(tenantID uuid.UUID, vesselID string) model.Provider[map[string]interface{}] {
	if m.VesselByIdProviderFunc != nil {
		return m.VesselByIdProviderFunc(tenantID, vesselID)
	}
	return func() (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
}

// AllVesselsProvider is a mock implementation
func (m *ProcessorMock) AllVesselsProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	if m.AllVesselsProviderFunc != nil {
		return m.AllVesselsProviderFunc(tenantID)
	}
	return func() ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
}

// CreateInstanceRoute is a mock implementation
func (m *ProcessorMock) CreateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error) {
	if m.CreateInstanceRouteFunc != nil {
		return m.CreateInstanceRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(route map[string]interface{}) (configuration.Model, error) {
		return func(route map[string]interface{}) (configuration.Model, error) {
			return configuration.Model{}, nil
		}
	}
}

// CreateInstanceRouteAndEmit is a mock implementation
func (m *ProcessorMock) CreateInstanceRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (configuration.Model, error) {
	if m.CreateInstanceRouteAndEmitFunc != nil {
		return m.CreateInstanceRouteAndEmitFunc(tenantID, route)
	}
	return configuration.Model{}, nil
}

// UpdateInstanceRoute is a mock implementation
func (m *ProcessorMock) UpdateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
	if m.UpdateInstanceRouteFunc != nil {
		return m.UpdateInstanceRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
		return func(routeID string) func(route map[string]interface{}) (configuration.Model, error) {
			return func(route map[string]interface{}) (configuration.Model, error) {
				return configuration.Model{}, nil
			}
		}
	}
}

// UpdateInstanceRouteAndEmit is a mock implementation
func (m *ProcessorMock) UpdateInstanceRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (configuration.Model, error) {
	if m.UpdateInstanceRouteAndEmitFunc != nil {
		return m.UpdateInstanceRouteAndEmitFunc(tenantID, routeID, route)
	}
	return configuration.Model{}, nil
}

// DeleteInstanceRoute is a mock implementation
func (m *ProcessorMock) DeleteInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	if m.DeleteInstanceRouteFunc != nil {
		return m.DeleteInstanceRouteFunc(mb)
	}
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			return nil
		}
	}
}

// DeleteInstanceRouteAndEmit is a mock implementation
func (m *ProcessorMock) DeleteInstanceRouteAndEmit(tenantID uuid.UUID, routeID string) error {
	if m.DeleteInstanceRouteAndEmitFunc != nil {
		return m.DeleteInstanceRouteAndEmitFunc(tenantID, routeID)
	}
	return nil
}

// GetInstanceRouteById is a mock implementation
func (m *ProcessorMock) GetInstanceRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error) {
	if m.GetInstanceRouteByIdFunc != nil {
		return m.GetInstanceRouteByIdFunc(tenantID, routeID)
	}
	return map[string]interface{}{}, nil
}

// GetAllInstanceRoutes is a mock implementation
func (m *ProcessorMock) GetAllInstanceRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if m.GetAllInstanceRoutesFunc != nil {
		return m.GetAllInstanceRoutesFunc(tenantID)
	}
	return []map[string]interface{}{}, nil
}

// InstanceRouteByIdProvider is a mock implementation
func (m *ProcessorMock) InstanceRouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	if m.InstanceRouteByIdProviderFunc != nil {
		return m.InstanceRouteByIdProviderFunc(tenantID, routeID)
	}
	return func() (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
}

// AllInstanceRoutesProvider is a mock implementation
func (m *ProcessorMock) AllInstanceRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	if m.AllInstanceRoutesProviderFunc != nil {
		return m.AllInstanceRoutesProviderFunc(tenantID)
	}
	return func() ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
}

// CreateMtsConfig is a mock implementation
func (m *ProcessorMock) CreateMtsConfig(mb *message.Buffer) func(tenantID uuid.UUID) func(config map[string]interface{}) (configuration.Model, error) {
	if m.CreateMtsConfigFunc != nil {
		return m.CreateMtsConfigFunc(mb)
	}
	return func(tenantID uuid.UUID) func(config map[string]interface{}) (configuration.Model, error) {
		return func(config map[string]interface{}) (configuration.Model, error) {
			return configuration.Model{}, nil
		}
	}
}

// CreateMtsConfigAndEmit is a mock implementation
func (m *ProcessorMock) CreateMtsConfigAndEmit(tenantID uuid.UUID, config map[string]interface{}) (configuration.Model, error) {
	if m.CreateMtsConfigAndEmitFunc != nil {
		return m.CreateMtsConfigAndEmitFunc(tenantID, config)
	}
	return configuration.Model{}, nil
}

// UpdateMtsConfig is a mock implementation
func (m *ProcessorMock) UpdateMtsConfig(mb *message.Buffer) func(tenantID uuid.UUID) func(configID string) func(config map[string]interface{}) (configuration.Model, error) {
	if m.UpdateMtsConfigFunc != nil {
		return m.UpdateMtsConfigFunc(mb)
	}
	return func(tenantID uuid.UUID) func(configID string) func(config map[string]interface{}) (configuration.Model, error) {
		return func(configID string) func(config map[string]interface{}) (configuration.Model, error) {
			return func(config map[string]interface{}) (configuration.Model, error) {
				return configuration.Model{}, nil
			}
		}
	}
}

// UpdateMtsConfigAndEmit is a mock implementation
func (m *ProcessorMock) UpdateMtsConfigAndEmit(tenantID uuid.UUID, configID string, config map[string]interface{}) (configuration.Model, error) {
	if m.UpdateMtsConfigAndEmitFunc != nil {
		return m.UpdateMtsConfigAndEmitFunc(tenantID, configID, config)
	}
	return configuration.Model{}, nil
}

// DeleteMtsConfig is a mock implementation
func (m *ProcessorMock) DeleteMtsConfig(mb *message.Buffer) func(tenantID uuid.UUID) func(configID string) error {
	if m.DeleteMtsConfigFunc != nil {
		return m.DeleteMtsConfigFunc(mb)
	}
	return func(tenantID uuid.UUID) func(configID string) error {
		return func(configID string) error {
			return nil
		}
	}
}

// DeleteMtsConfigAndEmit is a mock implementation
func (m *ProcessorMock) DeleteMtsConfigAndEmit(tenantID uuid.UUID, configID string) error {
	if m.DeleteMtsConfigAndEmitFunc != nil {
		return m.DeleteMtsConfigAndEmitFunc(tenantID, configID)
	}
	return nil
}

// GetMtsConfigById is a mock implementation
func (m *ProcessorMock) GetMtsConfigById(tenantID uuid.UUID, configID string) (map[string]interface{}, error) {
	if m.GetMtsConfigByIdFunc != nil {
		return m.GetMtsConfigByIdFunc(tenantID, configID)
	}
	return map[string]interface{}{}, nil
}

// GetAllMtsConfigs is a mock implementation
func (m *ProcessorMock) GetAllMtsConfigs(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if m.GetAllMtsConfigsFunc != nil {
		return m.GetAllMtsConfigsFunc(tenantID)
	}
	return []map[string]interface{}{}, nil
}

// MtsConfigByIdProvider is a mock implementation
func (m *ProcessorMock) MtsConfigByIdProvider(tenantID uuid.UUID, configID string) model.Provider[map[string]interface{}] {
	if m.MtsConfigByIdProviderFunc != nil {
		return m.MtsConfigByIdProviderFunc(tenantID, configID)
	}
	return func() (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
}

// AllMtsConfigsProvider is a mock implementation
func (m *ProcessorMock) AllMtsConfigsProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	if m.AllMtsConfigsProviderFunc != nil {
		return m.AllMtsConfigsProviderFunc(tenantID)
	}
	return func() ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
}

// SeedRoutes is a mock implementation
func (m *ProcessorMock) SeedRoutes(tenantID uuid.UUID) (configuration.SeedResult, error) {
	if m.SeedRoutesFunc != nil {
		return m.SeedRoutesFunc(tenantID)
	}
	return configuration.SeedResult{}, nil
}

// SeedInstanceRoutes is a mock implementation
func (m *ProcessorMock) SeedInstanceRoutes(tenantID uuid.UUID) (configuration.SeedResult, error) {
	if m.SeedInstanceRoutesFunc != nil {
		return m.SeedInstanceRoutesFunc(tenantID)
	}
	return configuration.SeedResult{}, nil
}

// SeedVessels is a mock implementation
func (m *ProcessorMock) SeedVessels(tenantID uuid.UUID) (configuration.SeedResult, error) {
	if m.SeedVesselsFunc != nil {
		return m.SeedVesselsFunc(tenantID)
	}
	return configuration.SeedResult{}, nil
}

// SeedRpsRewards is a mock implementation
func (m *ProcessorMock) SeedRpsRewards(tenantID uuid.UUID) (configuration.SeedResult, error) {
	if m.SeedRpsRewardsFunc != nil {
		return m.SeedRpsRewardsFunc(tenantID)
	}
	return configuration.SeedResult{}, nil
}

// CreateRpsReward is a mock implementation
func (m *ProcessorMock) CreateRpsReward(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsReward map[string]interface{}) (configuration.Model, error) {
	if m.CreateRpsRewardFunc != nil {
		return m.CreateRpsRewardFunc(mb)
	}
	return func(tenantID uuid.UUID) func(rpsReward map[string]interface{}) (configuration.Model, error) {
		return func(rpsReward map[string]interface{}) (configuration.Model, error) {
			return configuration.Model{}, nil
		}
	}
}

// CreateRpsRewardAndEmit is a mock implementation
func (m *ProcessorMock) CreateRpsRewardAndEmit(tenantID uuid.UUID, rpsReward map[string]interface{}) (configuration.Model, error) {
	if m.CreateRpsRewardAndEmitFunc != nil {
		return m.CreateRpsRewardAndEmitFunc(tenantID, rpsReward)
	}
	return configuration.Model{}, nil
}

// UpdateRpsReward is a mock implementation
func (m *ProcessorMock) UpdateRpsReward(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (configuration.Model, error) {
	if m.UpdateRpsRewardFunc != nil {
		return m.UpdateRpsRewardFunc(mb)
	}
	return func(tenantID uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (configuration.Model, error) {
		return func(rpsRewardID string) func(rpsReward map[string]interface{}) (configuration.Model, error) {
			return func(rpsReward map[string]interface{}) (configuration.Model, error) {
				return configuration.Model{}, nil
			}
		}
	}
}

// UpdateRpsRewardAndEmit is a mock implementation
func (m *ProcessorMock) UpdateRpsRewardAndEmit(tenantID uuid.UUID, rpsRewardID string, rpsReward map[string]interface{}) (configuration.Model, error) {
	if m.UpdateRpsRewardAndEmitFunc != nil {
		return m.UpdateRpsRewardAndEmitFunc(tenantID, rpsRewardID, rpsReward)
	}
	return configuration.Model{}, nil
}

// DeleteRpsReward is a mock implementation
func (m *ProcessorMock) DeleteRpsReward(mb *message.Buffer) func(tenantID uuid.UUID) func(rpsRewardID string) error {
	if m.DeleteRpsRewardFunc != nil {
		return m.DeleteRpsRewardFunc(mb)
	}
	return func(tenantID uuid.UUID) func(rpsRewardID string) error {
		return func(rpsRewardID string) error {
			return nil
		}
	}
}

// DeleteRpsRewardAndEmit is a mock implementation
func (m *ProcessorMock) DeleteRpsRewardAndEmit(tenantID uuid.UUID, rpsRewardID string) error {
	if m.DeleteRpsRewardAndEmitFunc != nil {
		return m.DeleteRpsRewardAndEmitFunc(tenantID, rpsRewardID)
	}
	return nil
}

// GetRpsRewardById is a mock implementation
func (m *ProcessorMock) GetRpsRewardById(tenantID uuid.UUID, rpsRewardID string) (map[string]interface{}, error) {
	if m.GetRpsRewardByIdFunc != nil {
		return m.GetRpsRewardByIdFunc(tenantID, rpsRewardID)
	}
	return map[string]interface{}{}, nil
}

// GetAllRpsRewards is a mock implementation
func (m *ProcessorMock) GetAllRpsRewards(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if m.GetAllRpsRewardsFunc != nil {
		return m.GetAllRpsRewardsFunc(tenantID)
	}
	return []map[string]interface{}{}, nil
}

// RpsRewardByIdProvider is a mock implementation
func (m *ProcessorMock) RpsRewardByIdProvider(tenantID uuid.UUID, rpsRewardID string) model.Provider[map[string]interface{}] {
	if m.RpsRewardByIdProviderFunc != nil {
		return m.RpsRewardByIdProviderFunc(tenantID, rpsRewardID)
	}
	return func() (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
}

// AllRpsRewardsProvider is a mock implementation
func (m *ProcessorMock) AllRpsRewardsProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	if m.AllRpsRewardsProviderFunc != nil {
		return m.AllRpsRewardsProviderFunc(tenantID)
	}
	return func() ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
}

// SeedMtsConfigs is a mock implementation
func (m *ProcessorMock) SeedMtsConfigs(tenantID uuid.UUID) (configuration.SeedResult, error) {
	if m.SeedMtsConfigsFunc != nil {
		return m.SeedMtsConfigsFunc(tenantID)
	}
	return configuration.SeedResult{}, nil
}
