package marriage

import (
	marriageMsg "atlas-marriages/kafka/message/marriage"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(marriageMsg.EnvCommandTopic), string(marriageMsg.EnvCommandTopic))
	_ = os.Setenv(string(marriageMsg.EnvEventTopicStatus), string(marriageMsg.EnvEventTopicStatus))
	// TestNewConfig exercises NewConfig with the literal test token
	// "test-token" rather than a manifest constant.
	_ = os.Setenv("test-token", "test-token")
	os.Exit(m.Run())
}
