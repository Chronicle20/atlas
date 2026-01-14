package mock

import (
	"atlas-npc-conversations/conversation/npc"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the npc.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement npc.Processor
	var _ npc.Processor = &ProcessorMock{}
}
