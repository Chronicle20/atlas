package mock

import (
	"atlas-saga-orchestrator/pending_change"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the pending_change.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	var _ pending_change.Processor = &ProcessorMock{}
}
