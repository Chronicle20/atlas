package storage

import (
	compartmentmsg "atlas-storage/kafka/message/compartment"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(compartmentmsg.EnvEventTopicStatus), string(compartmentmsg.EnvEventTopicStatus))
	os.Exit(m.Run())
}
