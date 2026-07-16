package asset

import (
	consumer2 "atlas-consumables/kafka/consumer"
	"atlas-consumables/kafka/message/asset"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// InitConsumers subscribes the service to the asset status topic. The reward
// flow registers a per-transaction once-handler on this topic (see
// consumable.ConsumeReward) to await the asset CREATED confirmation that marks
// a reward grant as successful; without this subscription that confirmation is
// never delivered and the box is never consumed.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}
