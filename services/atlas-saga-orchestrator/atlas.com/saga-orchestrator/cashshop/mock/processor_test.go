package mock

import (
	"atlas-saga-orchestrator/cashshop"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the cashshop.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement cashshop.Processor
	var _ cashshop.Processor = &ProcessorMock{}
}
