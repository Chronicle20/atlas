package drop

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
)

var ContinentDropsPath = "/drops/continents"

// LoadContinentDropFiles reads all JSON files from the continent drops directory
// and parses them into JSONModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadContinentDropFiles() ([]JSONModel, []error) {
	var models []JSONModel
	var errors []error

	entries, err := os.ReadDir(ContinentDropsPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read continent drops directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(ContinentDropsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var fileModels []JSONModel
		if err := json.Unmarshal(data, &fileModels); err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		models = append(models, fileModels...)
	}

	return models, errors
}

// DeleteAllForTenant deletes all continent drops for a specific tenant
func DeleteAllForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&entity{})
	return result.RowsAffected, result.Error
}
