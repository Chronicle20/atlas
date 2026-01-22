package npc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultConversationsPath = "/conversations/npc"

// getConversationsPath returns the path to the NPC conversations directory.
// It reads from the NPC_CONVERSATIONS_PATH environment variable, falling back
// to the default path if not set.
func getConversationsPath() string {
	if path := os.Getenv("NPC_CONVERSATIONS_PATH"); path != "" {
		return path
	}
	return defaultConversationsPath
}

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// LoadConversationFiles reads all JSON files from the conversations directory
// and parses them into RestModel structs. Returns the successfully parsed models
// and a slice of errors for any files that failed to load or parse.
func LoadConversationFiles() ([]RestModel, []error) {
	var models []RestModel
	var errors []error

	conversationsPath := getConversationsPath()
	entries, err := os.ReadDir(conversationsPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read conversations directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(conversationsPath, entry.Name())
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
