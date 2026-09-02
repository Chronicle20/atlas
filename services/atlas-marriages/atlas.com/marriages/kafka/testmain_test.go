package kafka

import (
	marriageMessage "atlas-marriages/kafka/message/marriage"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(marriageMessage.EnvCommandTopic), string(marriageMessage.EnvCommandTopic))
	_ = os.Setenv(string(marriageMessage.EnvEventTopicStatus), string(marriageMessage.EnvEventTopicStatus))
	// TestConsumerConfiguration exercises NewConfig with the literal test
	// token "test-token" rather than a manifest constant.
	_ = os.Setenv("test-token", "test-token")
	os.Exit(m.Run())
}
