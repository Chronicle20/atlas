package door

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// TestMain installs a no-op Kafka producer so handlers that emit (e.g.
// handleRemoved's buff.Cancel on the non-recast path) do not reach a real
// broker — without it those tests burn the producer's 10-retry/100ms→10s
// backoff and the suite takes ~17s instead of milliseconds (DOM-24).
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
