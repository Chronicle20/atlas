package mock

import (
	"atlas-npc-conversations/conversation/quest"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the quest.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement quest.Processor
	var _ quest.Processor = &ProcessorMock{}
}
