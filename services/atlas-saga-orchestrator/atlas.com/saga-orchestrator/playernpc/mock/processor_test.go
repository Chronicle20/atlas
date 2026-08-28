package mock

import (
	"atlas-saga-orchestrator/playernpc"
	"testing"
)

// TestProcessorMockImplementsProcessor verifies that ProcessorMock implements the playernpc.Processor interface
func TestProcessorMockImplementsProcessor(t *testing.T) {
	var _ playernpc.Processor = &ProcessorMock{}
}
