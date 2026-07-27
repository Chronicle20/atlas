package game_test

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// TestMain installs the shared no-op producer floor so any *AndEmit call
// that reaches the real producer.ProviderImpl (e.g. via the REST resource's
// StartAndEmit in resource_test.go) discards instead of hanging on broker
// retries. Mirrors kafka/consumer/rps/testmain_test.go.
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
