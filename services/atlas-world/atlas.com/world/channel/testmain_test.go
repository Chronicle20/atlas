package channel

import (
	"os"
	"testing"

	channel2 "atlas-world/kafka/message/channel"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(channel2.EnvCommandTopic), string(channel2.EnvCommandTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
