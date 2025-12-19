package mock

import (
	"atlas-npc-conversations/validation"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the validation.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement validation.Processor
	var _ validation.Processor = &ProcessorMock{}
}
