package context

import (
	"fmt"
	"regexp"
	"strings"
)

// contextPlaceholderRegex matches {context.xxx} patterns in text
var contextPlaceholderRegex = regexp.MustCompile(`\{context\.([a-zA-Z0-9_]+)\}`)

// ReplaceContextPlaceholders replaces all {context.xxx} placeholders in text with values from the context map
// Example: "You want {context.itemId}?" with context["itemId"]="2000002" -> "You want 2000002?"
func ReplaceContextPlaceholders(text string, ctx map[string]string) (string, error) {
	var missingKeys []string

	result := contextPlaceholderRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the key from {context.key}
		key := strings.TrimSuffix(strings.TrimPrefix(match, "{context."), "}")

		// Look up the value in the context map
		value, exists := ctx[key]
		if !exists {
			missingKeys = append(missingKeys, key)
			return match // Keep the placeholder if not found
		}

		return value
	})

	if len(missingKeys) > 0 {
		return result, fmt.Errorf("missing context keys: %v", missingKeys)
	}

	return result, nil
}

// ExtractContextValue extracts a context value from a string that may be in format:
// - "{context.xxx}" (with curly braces)
// - "context.xxx" (without curly braces)
// - "someValue" (literal value, not a context reference)
// Returns (value, isContextReference, error)
func ExtractContextValue(input string, ctx map[string]string) (string, bool, error) {
	// Check for {context.xxx} format
	if strings.HasPrefix(input, "{context.") && strings.HasSuffix(input, "}") {
		key := strings.TrimSuffix(strings.TrimPrefix(input, "{context."), "}")
		value, exists := ctx[key]
		if !exists {
			return "", true, fmt.Errorf("context key [%s] not found", key)
		}
		return value, true, nil
	}

	// Check for context.xxx format (legacy)
	if strings.HasPrefix(input, "context.") {
		key := strings.TrimPrefix(input, "context.")
		value, exists := ctx[key]
		if !exists {
			return "", true, fmt.Errorf("context key [%s] not found", key)
		}
		return value, true, nil
	}

	// Not a context reference, return as-is
	return input, false, nil
}
