package card

import (
	"atlas-monster-book/kafka/message/monsterbook"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(monsterbook.EnvEventTopicStatus), string(monsterbook.EnvEventTopicStatus))
	os.Exit(m.Run())
}
