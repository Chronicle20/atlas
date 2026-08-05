package configuration

import (
	"encoding/json"
	"errors"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateConfiguration creates a new configuration in the database
func CreateConfiguration(db *gorm.DB, e Entity) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&e).Error
	})
}

// UpdateConfiguration updates an existing configuration in the database
func UpdateConfiguration(db *gorm.DB, e Entity) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Save(&e).Error
	})
}

// DeleteConfiguration deletes a configuration from the database
func DeleteConfiguration(db *gorm.DB, tenantID uuid.UUID, resourceName string, resourceID string) error {
	var e Entity
	err := db.Where("tenant_id = ? AND resource_name = ?", tenantID, resourceName).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("configuration not found")
		}
		return err
	}

	// Parse the resource data to find and remove the specific resource by ID
	var resourceData map[string]interface{}
	if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
		return err
	}

	// For array of resources, filter out the one with matching ID
	if resources, ok := resourceData["data"].([]interface{}); ok {
		var newResources []interface{}
		found := false
		for _, resource := range resources {
			if resourceMap, ok := resource.(map[string]interface{}); ok {
				if id, ok := resourceMap["id"].(string); ok && id != resourceID {
					newResources = append(newResources, resource)
				} else if id == resourceID {
					found = true
				}
			}
		}

		if !found {
			return errors.New("resource not found")
		}

		resourceData["data"] = newResources
		updatedData, err := json.Marshal(resourceData)
		if err != nil {
			return err
		}

		e.ResourceData = updatedData
		return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			return tx.Save(&e).Error
		})
	}

	// If it's a single resource and the ID matches, delete the entire configuration
	if data, ok := resourceData["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok && id == resourceID {
			return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				return tx.Delete(&e).Error
			})
		}
	}

	return errors.New("resource not found")
}

// DeleteConfigurationByResourceName deletes all configuration rows for a tenant and resource name
func DeleteConfigurationByResourceName(db *gorm.DB, tenantID uuid.UUID, resourceName string) (int64, error) {
	result := db.Where("tenant_id = ? AND resource_name = ?", tenantID, resourceName).Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// AppendConfigurationEntries appends entries to the single
// (tenant_id, resource_name) configuration row, creating the row when it
// does not exist yet. The stored shape is a JSON:API document whose
// "data" is an array of {id, type, attributes} objects — the same shape
// CreateRoute/CreateVessel/CreateInstanceRoute produce, so a seeded row
// is indistinguishable from a hand-created one.
func AppendConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string, entries []map[string]interface{}) error {
	if len(entries) == 0 {
		return nil
	}
	existing, err := GetByTenantIdAndResourceNameProvider(tenantID, resourceName)(db)()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		payload, mErr := json.Marshal(map[string]interface{}{"data": entries})
		if mErr != nil {
			return mErr
		}
		return CreateConfiguration(db, Entity{
			ID:           uuid.New(),
			TenantId:     tenantID,
			ResourceName: resourceName,
			ResourceData: payload,
		})
	}
	if err != nil {
		return err
	}

	var document map[string]interface{}
	if err := json.Unmarshal(existing.ResourceData, &document); err != nil {
		return err
	}
	var merged []interface{}
	switch data := document["data"].(type) {
	case []interface{}:
		merged = data
	case map[string]interface{}:
		// Legacy single-object shape: promote it to an array so the
		// append below is uniform.
		merged = []interface{}{data}
	}
	for _, e := range entries {
		merged = append(merged, e)
	}
	document["data"] = merged

	payload, err := json.Marshal(document)
	if err != nil {
		return err
	}
	existing.ResourceData = payload
	return UpdateConfiguration(db, existing)
}

// CountConfigurationEntries returns the number of entries stored in the
// tenant's configuration row for resourceName, along with the row's
// updated_at. A missing row is (0, nil, nil) — not an error — because an
// unseeded tenant is a normal state the status endpoint must report.
func CountConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string) (int64, *time.Time, error) {
	existing, err := GetByTenantIdAndResourceNameProvider(tenantID, resourceName)(db)()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	var document map[string]interface{}
	if err := json.Unmarshal(existing.ResourceData, &document); err != nil {
		return 0, nil, err
	}
	updatedAt := existing.UpdatedAt
	switch data := document["data"].(type) {
	case []interface{}:
		return int64(len(data)), &updatedAt, nil
	case map[string]interface{}:
		return 1, &updatedAt, nil
	default:
		return 0, &updatedAt, nil
	}
}
