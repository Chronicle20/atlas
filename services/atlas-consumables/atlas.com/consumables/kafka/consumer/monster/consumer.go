package monster

import (
	consumer2 "atlas-consumables/kafka/consumer"
	monsterMsg "atlas-consumables/kafka/message/monster"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers subscribes to the dedicated catch topic only. There are no
// persistent handlers here: catch outcomes are consumed by the per-attempt
// once-handler RequestCatchMonster registers.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_catch_event")(monsterMsg.EnvEventTopicCatch)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}
