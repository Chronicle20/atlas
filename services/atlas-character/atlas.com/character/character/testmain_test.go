package character

import (
	character2 "atlas-character/kafka/message/character"
	dropmsg "atlas-character/kafka/message/drop"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvCommandTopic), string(character2.EnvCommandTopic))
	_ = os.Setenv(string(character2.EnvEventTopicCharacterStatus), string(character2.EnvEventTopicCharacterStatus))
	_ = os.Setenv(string(dropmsg.EnvCommandTopic), string(dropmsg.EnvCommandTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
