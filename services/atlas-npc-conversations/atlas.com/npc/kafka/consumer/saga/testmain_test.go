package saga

import (
	"atlas-npc-conversations/kafka/message/character"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character.EnvEventTopicCharacterStatus), string(character.EnvEventTopicCharacterStatus))
	os.Exit(m.Run())
}
