package seed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RoutesPath is the default path to the route seed data directory
var RoutesPath = "/routes"

// LoadRouteFiles reads all JSON files from the routes directory
// and parses them into JSONModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadRouteFiles() ([]JSONModel, []error) {
	var models []JSONModel
	var errors []error

	entries, err := os.ReadDir(RoutesPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read routes directory: %w", err)}
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

		filePath := filepath.Join(RoutesPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var route JSONModel
		if err := json.Unmarshal(data, &route); err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		models = append(models, route)
	}

	return models, errors
}
