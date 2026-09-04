package environment

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// TestMain installs a noop producer writer for the package so
// handleSetEnvironmentInMap / handleResetEnvironmentInMap's produce calls
// return instantly instead of retrying against an unreachable broker for
// ~42s per call.
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
