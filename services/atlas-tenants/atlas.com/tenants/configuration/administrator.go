package configuration

import (
	"atlas-tenants/database"
	"encoding/json"
	"errors"
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
