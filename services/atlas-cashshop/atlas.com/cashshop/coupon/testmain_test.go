package coupon

import (
	kafkacashshop "atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/message/item"
	"atlas-cashshop/kafka/message/wallet"
	"os"
	"testing"
)

// TestMain sets every topic token this package's tests rely on to its own
// name, so topic.EnvProvider resolves to the same literal the pre-existing
// assertions were already written against.
func TestMain(m *testing.M) {
	_ = os.Setenv(string(kafkacashshop.EnvEventTopicStatus), string(kafkacashshop.EnvEventTopicStatus))
	_ = os.Setenv(string(item.EnvStatusTopic), string(item.EnvStatusTopic))
	_ = os.Setenv(string(wallet.EnvEventTopicStatus), string(wallet.EnvEventTopicStatus))
	os.Exit(m.Run())
}
