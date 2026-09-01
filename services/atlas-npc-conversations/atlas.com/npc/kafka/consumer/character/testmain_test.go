package character

import (
	character2 "atlas-npc-conversations/kafka/message/character"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvCommandTopic), string(character2.EnvCommandTopic))
	_ = os.Setenv(string(character2.EnvEventTopicCharacterStatus), string(character2.EnvEventTopicCharacterStatus))
	os.Exit(m.Run())
}
