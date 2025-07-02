package mock

import (
	"atlas-npc-conversations/conversation"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the conversation.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	// This test will fail to compile if ProcessorMock doesn't implement conversation.Processor
	var _ conversation.Processor = &ProcessorMock{}
}