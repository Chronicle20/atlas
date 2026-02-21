package saga

import (
	sharedsaga "github.com/Chronicle20/atlas-saga"
)

// Re-export Builder from atlas-saga shared library
type Builder = sharedsaga.Builder

// NewBuilder creates a new Builder instance with default values
// Re-exported from atlas-saga shared library
var NewBuilder = sharedsaga.NewBuilder
