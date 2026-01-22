package conversation

import (
	scriptctx "github.com/Chronicle20/atlas-script-core/context"
)

// Re-export functions from atlas-script-core/context
var (
	// ReplaceContextPlaceholders replaces all {context.xxx} placeholders in text with values from the context map
	// Example: "You want {context.itemId}?" with context["itemId"]="2000002" -> "You want 2000002?"
	ReplaceContextPlaceholders = scriptctx.ReplaceContextPlaceholders

	// ExtractContextValue extracts a context value from a string that may be in format:
	// - "{context.xxx}" (with curly braces)
	// - "context.xxx" (without curly braces)
	// - "someValue" (literal value, not a context reference)
	// Returns (value, isContextReference, error)
	ExtractContextValue = scriptctx.ExtractContextValue
)
