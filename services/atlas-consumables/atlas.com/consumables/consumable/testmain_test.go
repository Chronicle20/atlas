package consumable

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// emitted captures everything this package's tests produce to Kafka. Installed
// once for the package; individual tests call emitted.Reset() rather than
// reinstalling the manager.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
