package marriage

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(EnvCommandTopic), string(EnvCommandTopic))
	_ = os.Setenv(string(EnvEventTopicStatus), string(EnvEventTopicStatus))
	os.Exit(m.Run())
}
