package character

import (
	"atlas-effective-stats/character"
	consumer2 "atlas-effective-stats/kafka/consumer"
	character2 "atlas-effective-stats/kafka/message/character"
	"atlas-effective-stats/stat"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatChanged)))
	}
}

func handleStatChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
	if e.Type != character2.StatusEventTypeStatChanged {
		return
	}

	// Skip if no values provided (old events without values)
	if e.Body.Values == nil || len(e.Body.Values) == 0 {
		return
	}

	// Check if this event contains stats we care about (MaxHP, MaxMP, primary stats)
	hasRelevantStats := false
	for _, update := range e.Body.Updates {
		switch update {
		case "MAX_HP", "MAX_MP", "STRENGTH", "DEXTERITY", "INTELLIGENCE", "LUCK":
			hasRelevantStats = true
			break
		}
	}

	if !hasRelevantStats {
		return
	}

	l.Debugf("Processing stat changed event for character [%d] with values: %v", e.CharacterId, e.Body.Values)

	p := character.NewProcessor(l, ctx)

	// Extract base stat values from the event
	base := extractBaseStats(e.Body.Values)

	// Update base stats in the registry
	ch := channel.NewModel(e.WorldId, e.Body.ChannelId)
	if err := p.SetBaseStats(ch, e.CharacterId, base); err != nil {
		l.WithError(err).Errorf("Unable to set base stats for character [%d].", e.CharacterId)
	}
}

// extractBaseStats builds a stat.Base from the values map
func extractBaseStats(values map[string]interface{}) stat.Base {
	return stat.NewBase(
		extractUint16(values, "strength"),
		extractUint16(values, "dexterity"),
		extractUint16(values, "luck"),
		extractUint16(values, "intelligence"),
		extractUint16(values, "max_hp"),
		extractUint16(values, "max_mp"),
	)
}

// extractUint16 safely extracts a uint16 from a map value
func extractUint16(values map[string]interface{}, key string) uint16 {
	v, ok := values[key]
	if !ok {
		return 0
	}

	switch val := v.(type) {
	case float64:
		return uint16(val)
	case int:
		return uint16(val)
	case int32:
		return uint16(val)
	case int64:
		return uint16(val)
	case uint16:
		return val
	case uint32:
		return uint16(val)
	case uint64:
		return uint16(val)
	default:
		return 0
	}
}
