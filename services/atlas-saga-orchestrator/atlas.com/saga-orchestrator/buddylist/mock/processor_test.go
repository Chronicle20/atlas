package mock

import (
	"atlas-saga-orchestrator/buddylist"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the buddylist.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	var _ buddylist.Processor = &ProcessorMock{}
}
