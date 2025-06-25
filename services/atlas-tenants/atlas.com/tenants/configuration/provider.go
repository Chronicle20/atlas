package configuration

import (
	"atlas-tenants/database"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetByTenantIdAndResourceNameProvider returns a provider for a configuration by tenant ID and resource name
func GetByTenantIdAndResourceNameProvider(tenantID uuid.UUID, resourceName string) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, map[string]interface{}{
			"tenant_id":     tenantID,
			"resource_name": resourceName,
		})
	}
}

// GetByTenantIdProvider returns a provider for all configurations for a tenant
func GetByTenantIdProvider(tenantID uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{
			"tenant_id": tenantID,
		})
	}
}

// GetRouteByIdProvider returns a provider for a specific route by ID
func GetRouteByIdProvider(tenantID uuid.UUID, routeID string) func(db *gorm.DB) model.Provider[map[string]interface{}] {
	return func(db *gorm.DB) model.Provider[map[string]interface{}] {
		entityProvider := GetByTenantIdAndResourceNameProvider(tenantID, "routes")(db)
		return model.Map(func(e Entity) (map[string]interface{}, error) {
			var resourceData map[string]interface{}
			if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
				return nil, err
			}

			// Check if it's an array of resources
			if resources, ok := resourceData["data"].([]interface{}); ok {
				for _, resource := range resources {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						if id, ok := resourceMap["id"].(string); ok && id == routeID {
							return resourceMap, nil
						}
					}
				}
				return nil, gorm.ErrRecordNotFound
			}

			// Check if it's a single resource
			if data, ok := resourceData["data"].(map[string]interface{}); ok {
				if id, ok := data["id"].(string); ok && id == routeID {
					return data, nil
				}
			}

			return nil, gorm.ErrRecordNotFound
		})(entityProvider)
	}
}

// GetAllRoutesProvider returns a provider for all routes for a tenant
func GetAllRoutesProvider(tenantID uuid.UUID) func(db *gorm.DB) model.Provider[[]map[string]interface{}] {
	return func(db *gorm.DB) model.Provider[[]map[string]interface{}] {
		entityProvider := GetByTenantIdAndResourceNameProvider(tenantID, "routes")(db)
		return model.Map(func(e Entity) ([]map[string]interface{}, error) {
			var resourceData map[string]interface{}
			if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
				return nil, err
			}

			// Check if it's an array of resources
			if resources, ok := resourceData["data"].([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(resources))
				for _, resource := range resources {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						result = append(result, resourceMap)
					}
				}
				return result, nil
			}

			// Check if it's a single resource
			if data, ok := resourceData["data"].(map[string]interface{}); ok {
				return []map[string]interface{}{data}, nil
			}

			return []map[string]interface{}{}, nil
		})(entityProvider)
	}
}

// GetVesselByIdProvider returns a provider for a specific vessel by ID
func GetVesselByIdProvider(tenantID uuid.UUID, vesselID string) func(db *gorm.DB) model.Provider[map[string]interface{}] {
	return func(db *gorm.DB) model.Provider[map[string]interface{}] {
		entityProvider := GetByTenantIdAndResourceNameProvider(tenantID, "vessels")(db)
		return model.Map(func(e Entity) (map[string]interface{}, error) {
			var resourceData map[string]interface{}
			if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
				return nil, err
			}

			// Check if it's an array of resources
			if resources, ok := resourceData["data"].([]interface{}); ok {
				for _, resource := range resources {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						if id, ok := resourceMap["id"].(string); ok && id == vesselID {
							return resourceMap, nil
						}
					}
				}
				return nil, gorm.ErrRecordNotFound
			}

			// Check if it's a single resource
			if data, ok := resourceData["data"].(map[string]interface{}); ok {
				if id, ok := data["id"].(string); ok && id == vesselID {
					return data, nil
				}
			}

			return nil, gorm.ErrRecordNotFound
		})(entityProvider)
	}
}

// GetAllVesselsProvider returns a provider for all vessels for a tenant
func GetAllVesselsProvider(tenantID uuid.UUID) func(db *gorm.DB) model.Provider[[]map[string]interface{}] {
	return func(db *gorm.DB) model.Provider[[]map[string]interface{}] {
		entityProvider := GetByTenantIdAndResourceNameProvider(tenantID, "vessels")(db)
		return model.Map(func(e Entity) ([]map[string]interface{}, error) {
			var resourceData map[string]interface{}
			if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
				return nil, err
			}

			// Check if it's an array of resources
			if resources, ok := resourceData["data"].([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(resources))
				for _, resource := range resources {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						result = append(result, resourceMap)
					}
				}
				return result, nil
			}

			// Check if it's a single resource
			if data, ok := resourceData["data"].(map[string]interface{}); ok {
				return []map[string]interface{}{data}, nil
			}

			return []map[string]interface{}{}, nil
		})(entityProvider)
	}
}
