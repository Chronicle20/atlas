package guild

import (
	guild2 "atlas-guilds/kafka/message/guild"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(guild2.EnvStatusEventTopic), string(guild2.EnvStatusEventTopic))
	os.Exit(m.Run())
}
