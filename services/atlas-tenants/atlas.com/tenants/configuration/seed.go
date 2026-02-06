package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultRoutesPath = "/configurations/routes"
const defaultInstanceRoutesPath = "/configurations/instance-routes"
const defaultVesselsPath = "/configurations/vessels"

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// getRoutesPath returns the path to the routes seed directory.
func getRoutesPath() string {
	if path := os.Getenv("ROUTES_SEED_PATH"); path != "" {
		return path
	}
	return defaultRoutesPath
}

// getInstanceRoutesPath returns the path to the instance routes seed directory.
func getInstanceRoutesPath() string {
	if path := os.Getenv("INSTANCE_ROUTES_SEED_PATH"); path != "" {
		return path
	}
	return defaultInstanceRoutesPath
}

// LoadRouteFiles reads all JSON files from the routes seed directory
// and parses them into map[string]interface{} structs.
func LoadRouteFiles() ([]map[string]interface{}, []error) {
	return loadSeedFiles(getRoutesPath())
}

// LoadInstanceRouteFiles reads all JSON files from the instance routes seed directory
// and parses them into map[string]interface{} structs.
func LoadInstanceRouteFiles() ([]map[string]interface{}, []error) {
	return loadSeedFiles(getInstanceRoutesPath())
}

// getVesselsPath returns the path to the vessels seed directory.
func getVesselsPath() string {
	if path := os.Getenv("VESSELS_SEED_PATH"); path != "" {
		return path
	}
	return defaultVesselsPath
}

// LoadVesselFiles reads all JSON files from the vessels seed directory
// and parses them into map[string]interface{} structs.
func LoadVesselFiles() ([]map[string]interface{}, []error) {
	return loadSeedFiles(getVesselsPath())
}

// loadSeedFiles reads all JSON files from the given directory and parses them.
func loadSeedFiles(dirPath string) ([]map[string]interface{}, []error) {
	var models []map[string]interface{}
	var errs []error

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read seed directory %s: %w", dirPath, err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var model map[string]interface{}
		if err := json.Unmarshal(data, &model); err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		models = append(models, model)
	}

	return models, errs
}
