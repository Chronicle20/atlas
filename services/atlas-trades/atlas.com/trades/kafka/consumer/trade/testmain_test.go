package trade

import (
	"os"
	"testing"

	trademsg "atlas-trades/kafka/message/trade"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(trademsg.EnvEventTopicStatus), string(trademsg.EnvEventTopicStatus))
	os.Exit(m.Run())
}
