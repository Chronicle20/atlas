package trade

import (
	"os"
	"testing"

	invitemsg "atlas-trades/kafka/message/invite"
	sagamsg "atlas-trades/kafka/message/saga"
	trademsg "atlas-trades/kafka/message/trade"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(trademsg.EnvEventTopicStatus), string(trademsg.EnvEventTopicStatus))
	_ = os.Setenv(string(trademsg.EnvCommandTopic), string(trademsg.EnvCommandTopic))
	_ = os.Setenv(string(invitemsg.EnvCommandTopic), string(invitemsg.EnvCommandTopic))
	_ = os.Setenv(string(sagamsg.EnvCommandTopic), string(sagamsg.EnvCommandTopic))
	os.Exit(m.Run())
}
