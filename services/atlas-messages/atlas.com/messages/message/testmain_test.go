package message

import (
	"atlas-messages/kafka/message/message"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(message.EnvEventTopicChat), string(message.EnvEventTopicChat))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
