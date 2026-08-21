package movement

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// sharedCapture is the package-wide producer stub. TeleportCharacter and
// ForCharacter both fire their Kafka emit on an unawaited goroutine
// (movement/processor.go), so a test's own async write can still be
// in flight when the next test starts. producertest.InstallCapturing (like
// InstallNoop) resets the producer manager singleton, and that reset is not
// safe to race against a concurrent read from a still-running prior test's
// goroutine -- so it must happen exactly once for the whole package, here,
// before any test (and therefore any goroutine) exists. Individual tests
// that need to assert on what was produced use sharedCapture directly
// instead of installing their own stub mid-run.
var sharedCapture *producertest.Capture

func TestMain(m *testing.M) {
	sharedCapture = producertest.InstallCapturing()
	os.Exit(m.Run())
}
