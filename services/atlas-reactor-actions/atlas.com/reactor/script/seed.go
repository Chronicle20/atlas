package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ScriptsPath = "/scripts/reactors"

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// LoadReactorScriptFiles reads all JSON files from the scripts directory
// and parses them into ReactorScript models. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadReactorScriptFiles() ([]ReactorScript, []error) {
	var scripts []ReactorScript
	var errors []error

	scriptsDir := os.Getenv("REACTOR_ACTIONS_DIR")
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

		var jsonScript jsonReactorScript
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

// convertJsonToModel converts a JSON reactor script to a domain model
func convertJsonToModel(js jsonReactorScript) (ReactorScript, error) {
	builder := NewReactorScriptBuilder().
		SetReactorId(js.ReactorId).
		SetDescription(js.Description)

	for _, jr := range js.HitRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return ReactorScript{}, fmt.Errorf("failed to convert hit rule [%s]: %w", jr.Id, err)
		}
		builder.AddHitRule(rule)
	}

	for _, jr := range js.ActRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return ReactorScript{}, fmt.Errorf("failed to convert act rule [%s]: %w", jr.Id, err)
		}
		builder.AddActRule(rule)
	}

	return builder.Build(), nil
}
