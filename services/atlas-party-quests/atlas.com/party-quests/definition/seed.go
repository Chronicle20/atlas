package definition

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultDefinitionsPath = "/party-quests"

func getDefinitionsPath() string {
	if path := os.Getenv("PARTY_QUEST_DEFINITIONS_PATH"); path != "" {
		return path
	}
	return defaultDefinitionsPath
}

type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

func LoadDefinitionFiles() ([]RestModel, []error) {
	var models []RestModel
	var errors []error

	definitionsPath := getDefinitionsPath()
	entries, err := os.ReadDir(definitionsPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read definitions directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(definitionsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var model RestModel
		if err := json.Unmarshal(data, &model); err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		models = append(models, model)
	}

	return models, errors
}
