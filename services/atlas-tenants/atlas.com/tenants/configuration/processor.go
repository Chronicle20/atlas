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
	// Create creates a new route configuration
	Create(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error)
	// CreateAndEmit creates a new route configuration and emits events
	CreateAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error)
	// Update updates an existing route configuration
	Update(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error)
	// UpdateAndEmit updates an existing route configuration and emits events
	UpdateAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error)
	// Delete deletes a route configuration
	Delete(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error
	// DeleteAndEmit deletes a route configuration and emits events
	DeleteAndEmit(tenantID uuid.UUID, routeID string) error
	// GetRouteById gets a route by ID
	GetRouteById(tenantID uuid.UUID, routeID string) (map[string]interface{}, error)
	// GetAllRoutes gets all routes for a tenant
	GetAllRoutes(tenantID uuid.UUID) ([]map[string]interface{}, error)
	// RouteByIdProvider returns a provider for a route by ID
	RouteByIdProvider(tenantID uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	// AllRoutesProvider returns a provider for all routes for a tenant
	AllRoutesProvider(tenantID uuid.UUID) model.Provider[[]map[string]interface{}]
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
func (p *ProcessorImpl) Create(mb *message.Buffer) func(tenantID uuid.UUID) func(route map[string]interface{}) (Model, error) {
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
					// Create a new array with the existing resource and the new one
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
func (p *ProcessorImpl) CreateAndEmit(tenantID uuid.UUID, route map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.Create(mb)(tenantID)(route)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
}

// Update updates an existing route configuration
func (p *ProcessorImpl) Update(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
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
func (p *ProcessorImpl) UpdateAndEmit(tenantID uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	mb := message.NewBuffer()
	result, err := p.Update(mb)(tenantID)(routeID)(route)
	if err != nil {
		return Model{}, err
	}

	// No events to emit for now

	return result, nil
}

// Delete deletes a route configuration
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(tenantID uuid.UUID) func(routeID string) error {
	return func(tenantID uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			return DeleteConfiguration(p.db, tenantID, "routes", routeID)
		}
	}
}

// DeleteAndEmit deletes a route configuration and emits events
func (p *ProcessorImpl) DeleteAndEmit(tenantID uuid.UUID, routeID string) error {
	mb := message.NewBuffer()
	err := p.Delete(mb)(tenantID)(routeID)
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
