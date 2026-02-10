package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultScriptsDir = "/scripts/map"
)

func getScriptsDir() string {
	if dir := os.Getenv("MAP_SCRIPTS_DIR"); dir != "" {
		return dir
	}
	return defaultScriptsDir
}

// SeedResult contains statistics from a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// LoadMapScriptFiles loads all map script JSON files from the scripts directory
func LoadMapScriptFiles() ([]MapScript, []error) {
	var scripts []MapScript
	var errors []error

	for _, scriptType := range []string{"onFirstUserEnter", "onUserEnter"} {
		dir := filepath.Join(getScriptsDir(), scriptType)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errors = append(errors, fmt.Errorf("failed to read directory %s: %w", dir, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			script, err := loadMapScriptFile(filePath, scriptType)
			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", filePath, err))
				continue
			}
			scripts = append(scripts, script)
		}
	}

	return scripts, errors
}

// loadMapScriptFile loads and parses a single map script JSON file
func loadMapScriptFile(filePath string, scriptType string) (MapScript, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return MapScript{}, fmt.Errorf("failed to read file: %w", err)
	}

	var jsonData jsonMapScript
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return MapScript{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	builder := NewMapScriptBuilder().
		SetScriptName(jsonData.ScriptName).
		SetScriptType(scriptType).
		SetDescription(jsonData.Description)

	for _, jr := range jsonData.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return MapScript{}, fmt.Errorf("failed to convert rule [%s]: %w", jr.Id, err)
		}
		builder.AddRule(rule)
	}

	return builder.Build(), nil
}
