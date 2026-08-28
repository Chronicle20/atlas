package pet_test

import (
	petmsg "atlas-pets/kafka/message/pet"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(petmsg.EnvCommandTopic), string(petmsg.EnvCommandTopic))
	_ = os.Setenv(string(petmsg.EnvCommandTopicMovement), string(petmsg.EnvCommandTopicMovement))
	os.Exit(m.Run())
}
