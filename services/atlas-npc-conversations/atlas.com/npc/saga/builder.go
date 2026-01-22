package saga

import (
	scriptsaga "github.com/Chronicle20/atlas-script-core/saga"
)

// Re-export Builder from atlas-script-core/saga
type Builder = scriptsaga.Builder

// NewBuilder creates a new Builder instance with default values
// Re-exported from atlas-script-core/saga
var NewBuilder = scriptsaga.NewBuilder
