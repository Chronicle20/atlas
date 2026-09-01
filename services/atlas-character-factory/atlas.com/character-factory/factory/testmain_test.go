package factory

import (
	"log"
	"os"
	"testing"

	sagaMessage "atlas-character-factory/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	if err := os.Setenv(string(sagaMessage.EnvCommandTopic), string(sagaMessage.EnvCommandTopic)); err != nil {
		log.Fatalf("failed to set %s: %v", sagaMessage.EnvCommandTopic, err)
	}
	producertest.InstallNoop()
	os.Exit(m.Run())
}
