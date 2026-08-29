package handler

import (
	"log"
	"os"
	"testing"

	messageCashShop "atlas-channel/kafka/message/cashshop"
	chalkboardMsg "atlas-channel/kafka/message/chalkboard"
	expression2 "atlas-channel/kafka/message/expression"
	kiteMsg "atlas-channel/kafka/message/kite"
	sagaMsg "atlas-channel/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

func TestMain(m *testing.M) {
	setEnv := func(k topic.Token) {
		if err := os.Setenv(string(k), string(k)); err != nil {
			log.Fatalf("failed to set %s: %v", k, err)
		}
	}
	setEnv(chalkboardMsg.EnvCommandTopic)
	setEnv(messageCashShop.EnvCommandTopic)
	setEnv(expression2.EnvExpressionCommand)
	setEnv(kiteMsg.EnvCommandTopic)
	setEnv(sagaMsg.EnvCommandTopic)
	os.Exit(m.Run())
}
