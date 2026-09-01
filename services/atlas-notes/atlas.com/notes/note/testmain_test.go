package note

import (
	notemsg "atlas-notes/kafka/message/note"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(notemsg.EnvEventTopicNoteStatus), string(notemsg.EnvEventTopicNoteStatus))
	producertest.InstallNoop()
	os.Exit(m.Run())
}
