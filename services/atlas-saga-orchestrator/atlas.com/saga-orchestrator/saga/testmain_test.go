package saga

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
