package character

import (
	"atlas-rates/character"
	consumer2 "atlas-rates/kafka/consumer"
	message "atlas-rates/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(message.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(message.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleMapChanged)))
	}
}

// handleMapChanged handles MAP_CHANGED events to initialize item tracking for characters
// This provides proactive initialization when characters change maps
func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent[message.StatusEventMapChangedBody]) {
	if e.Type != message.StatusEventTypeMapChanged {
		return
	}

	// Delegate to the shared initializer
	// This is idempotent - if already initialized, it returns immediately
	character.InitializeCharacterRates(l, ctx, e.CharacterId, e.WorldId, e.Body.ChannelId)
}
