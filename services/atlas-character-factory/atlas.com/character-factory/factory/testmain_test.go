package factory

import (
	"os"
	"testing"

	sagaMessage "atlas-character-factory/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	os.Setenv(string(sagaMessage.EnvCommandTopic), string(sagaMessage.EnvCommandTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
