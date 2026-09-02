package berserk

import (
	character2 "atlas-buffs/kafka/message/character"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvEventStatusTopic), string(character2.EnvEventStatusTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
