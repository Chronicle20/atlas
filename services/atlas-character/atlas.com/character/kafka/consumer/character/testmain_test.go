package character

import (
	character2 "atlas-character/kafka/message/character"
	"atlas-character/kafka/message/pending_change"
	sagamsg "atlas-character/kafka/message/saga"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvEventTopicCharacterStatus), string(character2.EnvEventTopicCharacterStatus))
	_ = os.Setenv(string(pending_change.EnvEventTopic), string(pending_change.EnvEventTopic))
	_ = os.Setenv(string(sagamsg.EnvCommandTopic), string(sagamsg.EnvCommandTopic))
	os.Exit(m.Run())
}
