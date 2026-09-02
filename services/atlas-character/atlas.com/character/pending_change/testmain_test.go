package pending_change

import (
	charactermsg "atlas-character/kafka/message/character"
	pendingchangemsg "atlas-character/kafka/message/pending_change"
	sagamsg "atlas-character/kafka/message/saga"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(pendingchangemsg.EnvEventTopic), string(pendingchangemsg.EnvEventTopic))
	_ = os.Setenv(string(sagamsg.EnvCommandTopic), string(sagamsg.EnvCommandTopic))
	_ = os.Setenv(string(charactermsg.EnvEventTopicCharacterStatus), string(charactermsg.EnvEventTopicCharacterStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
