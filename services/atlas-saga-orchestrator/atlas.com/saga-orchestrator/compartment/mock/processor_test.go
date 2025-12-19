package mock

import (
	"atlas-saga-orchestrator/compartment"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the compartment.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement compartment.Processor
	var _ compartment.Processor = &ProcessorMock{}
}