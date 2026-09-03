package conversation

import (
	"atlas-npc-conversations/kafka/message/npc"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(npc.EnvConversationCommandTopic), string(npc.EnvConversationCommandTopic))
	os.Exit(m.Run())
}
