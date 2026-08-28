package character

import (
	character2 "atlas-buffs/kafka/message/character"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// emitted records every message the package's tests produce, so the periodic
// tick tests can assert the exact CHANGE_HP amount. Capturing is a superset of
// the previous no-op install: tests that don't read it are unaffected. Install
// once per package (producer.Manager caches one writer per topic for the
// lifetime of the singleton); each test that reads it calls emitted.Reset()
// first.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvEventStatusTopic), string(character2.EnvEventStatusTopic))
	_ = os.Setenv(string(character2.EnvCommandTopicCharacter), string(character2.EnvCommandTopicCharacter))
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
