package configuration

import (
	"atlas-tenants/kafka/message"
	"context"
	"encoding/json"
	"errors"
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
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
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

				return Make(existing)
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

				return Make(entity)
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateAndEmit creates a new route configuration and emits events
func (p *ProcessorImpl) CreateRouteAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.CreateRoute(mb)(tenantID)(route)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
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

				return Make(existing)
			}
		}
	}
}

// UpdateAndEmit updates an existing route configuration and emits events
func (p *ProcessorImpl) UpdateRouteAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.UpdateRoute(mb)(tenantID)(routeID)(route)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
}

// Delete deletes a route configuration
func (p *ProcessorImpl) DeleteRoute(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			return DeleteConfiguration(p.db, tenantID, "routes", routeID)
		}
	}
}

// DeleteAndEmit deletes a route configuration and emits events
func (p *ProcessorImpl) DeleteRouteAndEmit(tenantID uuid.UUID, routeID string) error {
	mb := message.NewBuffer()
	err := p.DeleteRoute(mb)(tenantID)(routeID)
	if err != nil {
		return err
	}

	// No events to emit for now

	return nil
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

				return Make(existing)
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

				return Make(entity)
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateVesselAndEmit creates a new vessel configuration and emits events
func (p *ProcessorImpl) CreateVesselAndEmit(tenantID uuid.UUID, vessel map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.CreateVessel(mb)(tenantID)(vessel)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
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

				return Make(existing)
			}
		}
	}
}

// UpdateVesselAndEmit updates an existing vessel configuration and emits events
func (p *ProcessorImpl) UpdateVesselAndEmit(tenantID uuid.UUID, vesselID string, vessel map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.UpdateVessel(mb)(tenantID)(vesselID)(vessel)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
}

// DeleteVessel deletes a vessel configuration
func (p *ProcessorImpl) DeleteVessel(mb *message.Buffer) func(tenantID uuid.UUID) func(vesselID string) error {
	return func(tenantID uuid.UUID) func(vesselID string) error {
		return func(vesselID string) error {
			return DeleteConfiguration(p.db, tenantID, "vessels", vesselID)
		}
	}
}

// DeleteVesselAndEmit deletes a vessel configuration and emits events
func (p *ProcessorImpl) DeleteVesselAndEmit(tenantID uuid.UUID, vesselID string) error {
	mb := message.NewBuffer()
	err := p.DeleteVessel(mb)(tenantID)(vesselID)
	if err != nil {
		return err
	}

	// No events to emit for now

	return nil
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
