package drop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultMonsterDropsPath = "/drops/monsters"

// GetMonsterDropsPath returns the path to the monster drops directory,
// using the MONSTER_DROPS_PATH environment variable if set, otherwise
// falling back to the default path.
func GetMonsterDropsPath() string {
	if path := os.Getenv("MONSTER_DROPS_PATH"); path != "" {
		return path
	}
	return defaultMonsterDropsPath
}

// LoadMonsterDropFiles reads all JSON files from the monster drops directory
// and parses them into JSONModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadMonsterDropFiles() ([]JSONModel, []error) {
	var models []JSONModel
	var errors []error

	dropsPath := GetMonsterDropsPath()
	entries, err := os.ReadDir(dropsPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read monster drops directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dropsPath, entry.Name())
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

// DeleteAllForTenant deletes all monster drops for a specific tenant
func DeleteAllForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&entity{})
	return result.RowsAffected, result.Error
}
