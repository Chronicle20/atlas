package saga

import (
	"atlas-npc-conversations/kafka/message/npc"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(npc.EnvEventTopicCharacterStatus), string(npc.EnvEventTopicCharacterStatus))
	os.Exit(m.Run())
}
