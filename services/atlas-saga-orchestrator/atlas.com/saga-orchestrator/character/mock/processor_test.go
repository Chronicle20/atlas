package mock

import (
	"atlas-saga-orchestrator/character"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the character.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement character.Processor
	var _ character.Processor = &ProcessorMock{}
}