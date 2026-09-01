package thread

import (
	threadmsg "atlas-guilds/kafka/message/thread"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(threadmsg.EnvStatusEventTopic), string(threadmsg.EnvStatusEventTopic))
	os.Exit(m.Run())
}
