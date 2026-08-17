package mock

import (
	"atlas-saga-orchestrator/guild"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the guild.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	var _ guild.Processor = &ProcessorMock{}
}
