package monsterbook

import (
	charactermsg "atlas-monster-book/kafka/message/character"
	monsterbookmsg "atlas-monster-book/kafka/message/monsterbook"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(charactermsg.EnvEventTopicStatus), string(charactermsg.EnvEventTopicStatus))
	_ = os.Setenv(string(monsterbookmsg.EnvEventTopicStatus), string(monsterbookmsg.EnvEventTopicStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
