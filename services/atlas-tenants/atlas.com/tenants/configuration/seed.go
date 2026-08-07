package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultRpsRewardsPath = "/configurations/rps-rewards"
	defaultMtsConfigsPath = "/configurations/mts-configs"
)

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// getRpsRewardsPath returns the path to the rps-rewards seed directory.
func getRpsRewardsPath() string {
	if path := os.Getenv("RPS_REWARDS_SEED_PATH"); path != "" {
		return path
	}
	return defaultRpsRewardsPath
}

// LoadRpsRewardFiles reads all JSON files from the rps-rewards seed directory
// and parses them into map[string]interface{} structs.
func LoadRpsRewardFiles() ([]map[string]interface{}, []error) {
	return loadSeedFiles(getRpsRewardsPath())
}

// getMtsConfigsPath returns the path to the mts configs seed directory.
func getMtsConfigsPath() string {
	if path := os.Getenv("MTS_CONFIGS_SEED_PATH"); path != "" {
		return path
	}
	return defaultMtsConfigsPath
}

// LoadMtsConfigFiles reads all JSON files from the mts configs seed directory
// and parses them into map[string]interface{} structs.
func LoadMtsConfigFiles() ([]map[string]interface{}, []error) {
	return loadSeedFiles(getMtsConfigsPath())
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
