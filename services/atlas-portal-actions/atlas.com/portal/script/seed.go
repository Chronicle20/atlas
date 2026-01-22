package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ScriptsPath = "/scripts/portals"

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// LoadPortalScriptFiles reads all JSON files from the scripts directory
// and parses them into PortalScript models. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadPortalScriptFiles() ([]PortalScript, []error) {
	var scripts []PortalScript
	var errors []error

	scriptsDir := os.Getenv("PORTAL_SCRIPTS_DIR")
	if scriptsDir == "" {
		scriptsDir = ScriptsPath
	}

	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read scripts directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(scriptsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to read file: %w", entry.Name(), err))
			continue
		}

		var jsonScript jsonPortalScript
		if err := json.Unmarshal(data, &jsonScript); err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to parse JSON: %w", entry.Name(), err))
			continue
		}

		script, err := convertJsonToModel(jsonScript)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: failed to convert to model: %w", entry.Name(), err))
			continue
		}

		scripts = append(scripts, script)
	}

	return scripts, errors
}

// convertJsonToModel converts a JSON portal script to a domain model
func convertJsonToModel(js jsonPortalScript) (PortalScript, error) {
	builder := NewPortalScriptBuilder().
		SetPortalId(js.PortalId).
		SetMapId(js.MapId).
		SetDescription(js.Description)

	for _, jr := range js.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return PortalScript{}, fmt.Errorf("failed to convert rule [%s]: %w", jr.Id, err)
		}
		builder.AddRule(rule)
	}

	return builder.Build(), nil
}
