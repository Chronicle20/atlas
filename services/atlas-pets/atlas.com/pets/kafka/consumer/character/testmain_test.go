package character_test

import (
	character2 "atlas-pets/kafka/message/character"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(character2.EnvEventTopicCharacterStatus), string(character2.EnvEventTopicCharacterStatus))
	os.Exit(m.Run())
}
