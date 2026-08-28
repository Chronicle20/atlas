package handler

import (
	"os"
	"testing"

	messageCashShop "atlas-channel/kafka/message/cashshop"
	chalkboardMsg "atlas-channel/kafka/message/chalkboard"
	expression2 "atlas-channel/kafka/message/expression"
	kiteMsg "atlas-channel/kafka/message/kite"
	sagaMsg "atlas-channel/kafka/message/saga"
)

func TestMain(m *testing.M) {
	os.Setenv(string(chalkboardMsg.EnvCommandTopic), string(chalkboardMsg.EnvCommandTopic))
	os.Setenv(string(messageCashShop.EnvCommandTopic), string(messageCashShop.EnvCommandTopic))
	os.Setenv(string(expression2.EnvExpressionCommand), string(expression2.EnvExpressionCommand))
	os.Setenv(string(kiteMsg.EnvCommandTopic), string(kiteMsg.EnvCommandTopic))
	os.Setenv(string(sagaMsg.EnvCommandTopic), string(sagaMsg.EnvCommandTopic))
	os.Exit(m.Run())
}
