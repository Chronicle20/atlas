package mts

import (
	"atlas-mts/kafka/message/mts"
	msgsaga "atlas-mts/kafka/message/saga"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(mts.EnvStatusEventTopic), string(mts.EnvStatusEventTopic))
	_ = os.Setenv(string(msgsaga.EnvCommandTopic), string(msgsaga.EnvCommandTopic))
	os.Exit(m.Run())
}
