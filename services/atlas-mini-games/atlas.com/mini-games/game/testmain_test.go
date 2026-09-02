package game

import (
	"atlas-mini-games/kafka/message/minigame"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(minigame.EnvEventTopicStatus), string(minigame.EnvEventTopicStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
