package asset

import (
	consumer2 "atlas-consumables/kafka/consumer"
	"atlas-consumables/kafka/message/asset"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers subscribes the service to the asset status topic. The reward
// flow registers a per-transaction once-handler on this topic (see
// consumable.ConsumeReward) to await the asset CREATED confirmation that marks
// a reward grant as successful; without this subscription that confirmation is
// never delivered and the box is never consumed.
//
// Reads from the latest offset (mirrors atlas-channel's asset-status consumer):
// this topic is high-volume and the reward flow only cares about events that
// arrive after a request is in flight, so replaying history on start would be
// pure waste and could delay live confirmations behind the backlog.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}
