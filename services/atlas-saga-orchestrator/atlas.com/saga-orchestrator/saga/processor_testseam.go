//go:build test

package saga

import (
	"context"

	"github.com/sirupsen/logrus"
)

// SetProcessorFactoryForTest swaps the underlying Processor factory and returns
// the previous factory for restoration. Compiled only with -tags=test —
// production code cannot reach this seam. Used by integration tests that need
// to inject mock character/compartment processors to avoid real Kafka emissions.
func SetProcessorFactoryForTest(fn func(logger logrus.FieldLogger, ctx context.Context) Processor) func(logger logrus.FieldLogger, ctx context.Context) Processor {
	prev := newProcessorFn
	newProcessorFn = fn
	return prev
}
