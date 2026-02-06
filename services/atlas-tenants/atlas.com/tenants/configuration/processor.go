package configuration

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/kafka/producer"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor defines the interface for configuration operations
type Processor interface {
	// Route operations
	// CreateRoute creates a new route configuration
	CreateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error)
	// CreateRouteAndEmit creates a new route configuration and emits events
	CreateRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error)
	// UpdateRoute updates an existing route configuration
	UpdateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error)
	// UpdateRouteAndEmit updates an existing route configuration and emits events
	UpdateRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error)
	// DeleteRoute deletes a route configuration
	DeleteRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error
	// DeleteRouteAndEmit deletes a route configuration and emits events
	DeleteRouteAndEmit(tenantID uuid.UUID, routeID string) error
	// GetRouteById gets a route by ID
	GetRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error)
	// GetAllRoutes gets all routes for a tenant
	GetAllRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error)
	// RouteByIdProvider returns a provider for a route by ID
	RouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	// AllRoutesProvider returns a provider for all routes for a tenant
	AllRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// Instance route operations
	// CreateInstanceRoute creates a new instance route configuration
	CreateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error)
	// CreateInstanceRouteAndEmit creates a new instance route configuration and emits events
	CreateInstanceRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error)
	// UpdateInstanceRoute updates an existing instance route configuration
	UpdateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error)
	// UpdateInstanceRouteAndEmit updates an existing instance route configuration and emits events
	UpdateInstanceRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error)
	// DeleteInstanceRoute deletes an instance route configuration
	DeleteInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error
	// DeleteInstanceRouteAndEmit deletes an instance route configuration and emits events
	DeleteInstanceRouteAndEmit(tenantID uuid.UUID, routeID string) error
	// GetInstanceRouteById gets an instance route by ID
	GetInstanceRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error)
	// GetAllInstanceRoutes gets all instance routes for a tenant
	GetAllInstanceRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error)
	// InstanceRouteByIdProvider returns a provider for an instance route by ID
	InstanceRouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	// AllInstanceRoutesProvider returns a provider for all instance routes for a tenant
	AllInstanceRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// Vessel operations
	// CreateVessel creates a new vessel configuration
	CreateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vessel map[string]interface{}) (Model, error)
	// CreateVesselAndEmit creates a new vessel configuration and emits events
	CreateVesselAndEmit(tenantID uuid.UUID, vessel map[string]interface{}) (Model, error)
	// UpdateVessel updates an existing vessel configuration
	UpdateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error)
	// UpdateVesselAndEmit updates an existing vessel configuration and emits events
	UpdateVesselAndEmit(tenantID uuid.UUID, vesselID string, vessel map[string]interface{}) (Model, error)
	// DeleteVessel deletes a vessel configuration
	DeleteVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) error
	// DeleteVesselAndEmit deletes a vessel configuration and emits events
	DeleteVesselAndEmit(tenantID uuid.UUID, vesselID string) error
	// GetVesselById gets a vessel by ID
	GetVesselById(tenantID uuid.UUID, vesselID string) (map[string]interface{}, error)
	// GetAllVessels gets all vessels for a tenant
	GetAllVessels(tenantID uuid.UUID) ([]map[string]interface{}, error)
	// VesselByIdProvider returns a provider for a vessel by ID
	VesselByIdProvider(tenantID uuid.UUID, vesselID string) model.Provider[map[string]interface{}]
	// AllVesselsProvider returns a provider for all vessels for a tenant
	AllVesselsProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]

	// Seed operations
	// SeedRoutes clears existing routes for a tenant and loads them from seed files
	SeedRoutes(tenantID uuid.UUID) (SeedResult, error)
	// SeedInstanceRoutes clears existing instance routes for a tenant and loads them from seed files
	SeedInstanceRoutes(tenantID uuid.UUID) (SeedResult, error)
	// SeedVessels clears existing vessels for a tenant and loads them from seed files
	SeedVessels(tenantID uuid.UUID) (SeedResult, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	p   producer.Provider
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

// Create creates a new route configuration
func (p *ProcessorImpl) CreateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error) {
		return func(route map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "routes")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new route to the array
					resources = append(resources, route)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// CreateRoute a new array with the existing resource and the new one
					resourceData, err = CreateRouteJsonData([]map[string]interface{}{route})
					if err != nil {
						return Model{}, err
					}
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantID, EventTypeRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleRouteJsonData(route)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantID:     tenantID,
					ResourceName: "routes",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantID, EventTypeRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateRouteAndEmit creates a new route configuration and emits events
func (p *ProcessorImpl) CreateRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.CreateRoute(mb)(tenantID)(route)
		}
	})(tenantID)
}

// Update updates an existing route configuration
func (p *ProcessorImpl) UpdateRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
		return func(routeID string) func(route map[string]interface{}) (Model, error) {
			return func(route map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "routes")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the route ID matches
				route["id"] = routeID

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == routeID {
								resources[i] = route
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("route not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == routeID {
						existingData["data"] = route
					} else {
						return Model{}, errors.New("route not found")
					}
				} else {
					return Model{}, errors.New("invalid resource data format")
				}

				resourceData, err := json.Marshal(existingData)
				if err != nil {
					return Model{}, err
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantID, EventTypeRouteUpdated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateRouteAndEmit updates an existing route configuration and emits events
func (p *ProcessorImpl) UpdateRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.UpdateRoute(mb)(tenantID)(routeID)(route)
		}
	})(tenantID)
}

// Delete deletes a route configuration
func (p *ProcessorImpl) DeleteRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			if err := DeleteConfiguration(p.db, tenantID, "routes", routeID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantID, EventTypeRouteDeleted, routeID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteRouteAndEmit deletes a route configuration and emits events
func (p *ProcessorImpl) DeleteRouteAndEmit(tenantID uuid.UUID, routeID string) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.DeleteRoute(mb)(tenantID)(routeID)
	})
}

// GetRouteById gets a route by ID
func (p *ProcessorImpl) GetRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error) {
	return p.RouteByIdProvider(tenantID, routeID)()
}

// GetAllRoutes gets all routes for a tenant
func (p *ProcessorImpl) GetAllRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllRoutesProvider(tenantID)()
}

// RouteByIdProvider returns a provider for a route by ID
func (p *ProcessorImpl) RouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	return GetRouteByIdProvider(tenantID, routeID)(p.db)
}

// AllRoutesProvider returns a provider for all routes for a tenant
func (p *ProcessorImpl) AllRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllRoutesProvider(tenantID)(p.db)
}

// CreateVessel creates a new vessel configuration
func (p *ProcessorImpl) CreateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vessel map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(vessel map[string]interface{}) (Model, error) {
		return func(vessel map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "vessels")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new vessel to the array
					resources = append(resources, vessel)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// CreateRoute a new array with the existing resource and the new one
					resourceData, err = CreateVesselJsonData([]map[string]interface{}{vessel})
					if err != nil {
						return Model{}, err
					}
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				vesselID := ""
				if id, ok := vessel["id"].(string); ok {
					vesselID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantID, EventTypeVesselCreated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleVesselJsonData(vessel)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantID:     tenantID,
					ResourceName: "vessels",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				vesselID := ""
				if id, ok := vessel["id"].(string); ok {
					vesselID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantID, EventTypeVesselCreated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateVesselAndEmit creates a new vessel configuration and emits events
func (p *ProcessorImpl) CreateVesselAndEmit(tenantID uuid.UUID, vessel map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.CreateVessel(mb)(tenantID)(vessel)
		}
	})(tenantID)
}

// UpdateVessel updates an existing vessel configuration
func (p *ProcessorImpl) UpdateVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
		return func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
			return func(vessel map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "vessels")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the vessel ID matches
				vessel["id"] = vesselID

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == vesselID {
								resources[i] = vessel
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("vessel not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == vesselID {
						existingData["data"] = vessel
					} else {
						return Model{}, errors.New("vessel not found")
					}
				} else {
					return Model{}, errors.New("invalid resource data format")
				}

				resourceData, err := json.Marshal(existingData)
				if err != nil {
					return Model{}, err
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantID, EventTypeVesselUpdated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateVesselAndEmit updates an existing vessel configuration and emits events
func (p *ProcessorImpl) UpdateVesselAndEmit(tenantID uuid.UUID, vesselID string, vessel map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.UpdateVessel(mb)(tenantID)(vesselID)(vessel)
		}
	})(tenantID)
}

// DeleteVessel deletes a vessel configuration
func (p *ProcessorImpl) DeleteVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) error {
	return func(tenantID uuid.UUID) func(vesselID string) error {
		return func(vesselID string) error {
			if err := DeleteConfiguration(p.db, tenantID, "vessels", vesselID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantID, EventTypeVesselDeleted, vesselID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteVesselAndEmit deletes a vessel configuration and emits events
func (p *ProcessorImpl) DeleteVesselAndEmit(tenantID uuid.UUID, vesselID string) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.DeleteVessel(mb)(tenantID)(vesselID)
	})
}

// GetVesselById gets a vessel by ID
func (p *ProcessorImpl) GetVesselById(tenantID uuid.UUID, vesselID string) (map[string]interface{}, error) {
	return p.VesselByIdProvider(tenantID, vesselID)()
}

// GetAllVessels gets all vessels for a tenant
func (p *ProcessorImpl) GetAllVessels(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllVesselsProvider(tenantID)()
}

// VesselByIdProvider returns a provider for a vessel by ID
func (p *ProcessorImpl) VesselByIdProvider(tenantID uuid.UUID, vesselID string) model.Provider[map[string]interface{}] {
	return GetVesselByIdProvider(tenantID, vesselID)(p.db)
}

// AllVesselsProvider returns a provider for all vessels for a tenant
func (p *ProcessorImpl) AllVesselsProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllVesselsProvider(tenantID)(p.db)
}

// CreateInstanceRoute creates a new instance route configuration
func (p *ProcessorImpl) CreateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error) {
		return func(route map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "instance-routes")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				if resources, ok := existingData["data"].([]interface{}); ok {
					resources = append(resources, route)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					resourceData, err = CreateInstanceRouteJsonData([]map[string]interface{}{route})
					if err != nil {
						return Model{}, err
					}
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantID, EventTypeInstanceRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				resourceData, err = CreateSingleInstanceRouteJsonData(route)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantID:     tenantID,
					ResourceName: "instance-routes",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantID, EventTypeInstanceRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				return Model{}, err
			}
		}
	}
}

// CreateInstanceRouteAndEmit creates a new instance route configuration and emits events
func (p *ProcessorImpl) CreateInstanceRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.CreateInstanceRoute(mb)(tenantID)(route)
		}
	})(tenantID)
}

// UpdateInstanceRoute updates an existing instance route configuration
func (p *ProcessorImpl) UpdateInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
	return func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
		return func(routeID string) func(route map[string]interface{}) (Model, error) {
			return func(route map[string]interface{}) (Model, error) {
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantID, "instance-routes")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				route["id"] = routeID

				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == routeID {
								resources[i] = route
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("instance route not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == routeID {
						existingData["data"] = route
					} else {
						return Model{}, errors.New("instance route not found")
					}
				} else {
					return Model{}, errors.New("invalid resource data format")
				}

				resourceData, err := json.Marshal(existingData)
				if err != nil {
					return Model{}, err
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantID, EventTypeInstanceRouteUpdated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateInstanceRouteAndEmit updates an existing instance route configuration and emits events
func (p *ProcessorImpl) UpdateInstanceRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(tenantID uuid.UUID) (Model, error) {
			return p.UpdateInstanceRoute(mb)(tenantID)(routeID)(route)
		}
	})(tenantID)
}

// DeleteInstanceRoute deletes an instance route configuration
func (p *ProcessorImpl) DeleteInstanceRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			if err := DeleteConfiguration(p.db, tenantID, "instance-routes", routeID); err != nil {
				return err
			}

			if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantID, EventTypeInstanceRouteDeleted, routeID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteInstanceRouteAndEmit deletes an instance route configuration and emits events
func (p *ProcessorImpl) DeleteInstanceRouteAndEmit(tenantID uuid.UUID, routeID string) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.DeleteInstanceRoute(mb)(tenantID)(routeID)
	})
}

// GetInstanceRouteById gets an instance route by ID
func (p *ProcessorImpl) GetInstanceRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error) {
	return p.InstanceRouteByIdProvider(tenantID, routeID)()
}

// GetAllInstanceRoutes gets all instance routes for a tenant
func (p *ProcessorImpl) GetAllInstanceRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllInstanceRoutesProvider(tenantID)()
}

// InstanceRouteByIdProvider returns a provider for an instance route by ID
func (p *ProcessorImpl) InstanceRouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	return GetInstanceRouteByIdProvider(tenantID, routeID)(p.db)
}

// AllInstanceRoutesProvider returns a provider for all instance routes for a tenant
func (p *ProcessorImpl) AllInstanceRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllInstanceRoutesProvider(tenantID)(p.db)
}

// SeedRoutes clears existing routes for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedRoutes(tenantID uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding routes for tenant [%s]", tenantID)

	result := SeedResult{}

	// Delete all existing routes for this tenant
	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantID, "routes")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing routes: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load route files from the filesystem
	routes, loadErrors := LoadRouteFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each route
	for _, route := range routes {
		id, _ := route["id"].(string)
		_, err := p.CreateRouteAndEmit(tenantID, route)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Route seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantID, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// SeedInstanceRoutes clears existing instance routes for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedInstanceRoutes(tenantID uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding instance routes for tenant [%s]", tenantID)

	result := SeedResult{}

	// Delete all existing instance routes for this tenant
	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantID, "instance-routes")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing instance routes: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load instance route files from the filesystem
	routes, loadErrors := LoadInstanceRouteFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each instance route
	for _, route := range routes {
		id, _ := route["id"].(string)
		_, err := p.CreateInstanceRouteAndEmit(tenantID, route)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Instance route seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantID, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// SeedVessels clears existing vessels for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedVessels(tenantID uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding vessels for tenant [%s]", tenantID)

	result := SeedResult{}

	// Delete all existing vessels for this tenant
	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantID, "vessels")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing vessels: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load vessel files from the filesystem
	vessels, loadErrors := LoadVesselFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each vessel
	for _, vessel := range vessels {
		id, _ := vessel["id"].(string)
		_, err := p.CreateVesselAndEmit(tenantID, vessel)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Vessel seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantID, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}
