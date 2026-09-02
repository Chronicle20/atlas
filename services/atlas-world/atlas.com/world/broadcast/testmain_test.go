package broadcast

import (
	"os"
	"testing"

	bmessage "atlas-world/kafka/message/broadcast"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(bmessage.EnvEventTopicWorldBroadcastStatus), string(bmessage.EnvEventTopicWorldBroadcastStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
