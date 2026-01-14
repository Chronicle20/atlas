package shops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ShopsPath is the default path to the shop seed data directory
var ShopsPath = "/shops"

// LoadShopFiles reads all JSON files from the shops directory
// and parses them into JSONModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadShopFiles() ([]JSONModel, []error) {
	var models []JSONModel
	var errors []error

	entries, err := os.ReadDir(ShopsPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read shops directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Skip the schema file
		if entry.Name() == "schema.json" {
			continue
		}

		filePath := filepath.Join(ShopsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var shop JSONModel
		if err := json.Unmarshal(data, &shop); err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		models = append(models, shop)
	}

	return models, errors
}
