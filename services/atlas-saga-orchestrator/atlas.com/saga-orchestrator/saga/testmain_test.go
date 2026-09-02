package saga

import (
	incubatormsg "atlas-saga-orchestrator/kafka/message/incubator"
	npcmsg "atlas-saga-orchestrator/kafka/message/npc"
	playernpcmsg "atlas-saga-orchestrator/kafka/message/playernpc"
	sagamsg "atlas-saga-orchestrator/kafka/message/saga"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(incubatormsg.EnvEventTopicIncubatorResult), string(incubatormsg.EnvEventTopicIncubatorResult))
	_ = os.Setenv(string(npcmsg.EnvCommandTopic), string(npcmsg.EnvCommandTopic))
	_ = os.Setenv(string(sagamsg.EnvStatusEventTopic), string(sagamsg.EnvStatusEventTopic))
	_ = os.Setenv(string(playernpcmsg.EnvCommandTopic), string(playernpcmsg.EnvCommandTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
