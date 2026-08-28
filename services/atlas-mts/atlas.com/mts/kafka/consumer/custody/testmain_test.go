package custody

import (
	"atlas-mts/kafka/message/custody"
	mtsmsg "atlas-mts/kafka/message/mts"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(custody.EnvStatusEventTopic), string(custody.EnvStatusEventTopic))
	_ = os.Setenv(string(mtsmsg.EnvStatusEventTopic), string(mtsmsg.EnvStatusEventTopic))
	os.Exit(m.Run())
}
