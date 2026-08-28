package rps

import (
	rpsMsg "atlas-rps/kafka/message/rps"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// TestMain installs the shared no-op producer floor so any emit that escapes
// a test discards instead of hanging on broker retries. Individual tests that
// need to inspect emissions install their own capturing manager on top of
// this floor (see setupCapturingProducer in consumer_test.go).
func TestMain(m *testing.M) {
	_ = os.Setenv(string(rpsMsg.EnvCommandTopic), string(rpsMsg.EnvCommandTopic))
	_ = os.Setenv(string(rpsMsg.EnvEventTopic), string(rpsMsg.EnvEventTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
