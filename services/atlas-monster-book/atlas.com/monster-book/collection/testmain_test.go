package collection

import (
	"atlas-monster-book/kafka/message/monsterbook"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(monsterbook.EnvEventTopicStatus), string(monsterbook.EnvEventTopicStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
