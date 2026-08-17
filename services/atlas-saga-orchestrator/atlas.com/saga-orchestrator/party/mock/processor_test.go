package mock

import (
	"atlas-saga-orchestrator/party"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the party.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	var _ party.Processor = &ProcessorMock{}
}
