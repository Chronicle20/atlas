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

const MonsterDropsPath = "/drops/monsters"

// LoadMonsterDropFiles reads all JSON files from the monster drops directory
// and parses them into JSONModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadMonsterDropFiles() ([]JSONModel, []error) {
	var models []JSONModel
	var errors []error

	entries, err := os.ReadDir(MonsterDropsPath)
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

		filePath := filepath.Join(MonsterDropsPath, entry.Name())
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
