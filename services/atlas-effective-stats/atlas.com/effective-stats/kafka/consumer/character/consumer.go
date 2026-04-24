package character

import (
	"atlas-effective-stats/character"
	consumer2 "atlas-effective-stats/kafka/consumer"
	character2 "atlas-effective-stats/kafka/message/character"
	"atlas-effective-stats/stat"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatChanged))); err != nil {
			return err
		}
		return nil
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
	ch := channel.NewModel(e.WorldId, e.Body.ChannelId)

	// Fetch current base stats and merge. atlas-character emits STAT_CHANGED
	// with only the fields that changed (e.g. a single AP into luck sends
	// {luck: 38}), so treating Values as a full snapshot would zero every
	// absent field. That's how character 12 silently died: each LUCK AP
	// wiped MaxHP/MaxMP to 0 in the registry, and the next HoT regen tick
	// read MaxHP=0 through GetEffectiveStats, clamped HP to 0, and emitted
	// a DIED event with killerType=UNKNOWN.
	currentBase := lookupCurrentBase(ctx, l, ch, e.CharacterId)
	merged := mergeBaseStats(currentBase, e.Body.Values)

	if err := p.SetBaseStats(ch, e.CharacterId, merged); err != nil {
		l.WithError(err).Errorf("Unable to set base stats for character [%d].", e.CharacterId)
	}
}

// lookupCurrentBase returns the character's current base stats from the
// registry. If the character is not yet initialized, it triggers lazy
// initialization (which fetches a full snapshot from atlas-character) and
// re-reads. Returns the zero stat.Base only if the character cannot be
// located after init — in that case the partial event is effectively all
// the caller has, which is still better than unconditionally zeroing.
func lookupCurrentBase(ctx context.Context, l logrus.FieldLogger, ch channel.Model, characterId uint32) stat.Base {
	if m, err := character.GetRegistry().Get(ctx, characterId); err == nil {
		return m.BaseStats()
	}

	if err := character.InitializeCharacter(l, ctx, characterId, ch); err != nil {
		l.WithError(err).Warnf("Unable to initialize character [%d] before merging stat update; base will default to zero for absent fields.", characterId)
	}

	if m, err := character.GetRegistry().Get(ctx, characterId); err == nil {
		return m.BaseStats()
	}
	return stat.Base{}
}

// mergeBaseStats returns a new stat.Base with fields from current overridden
// only by fields explicitly present in values. Fields absent from values are
// preserved from current.
func mergeBaseStats(current stat.Base, values map[string]interface{}) stat.Base {
	return stat.NewBase(
		mergeUint16(current.Strength(), values, "strength"),
		mergeUint16(current.Dexterity(), values, "dexterity"),
		mergeUint16(current.Luck(), values, "luck"),
		mergeUint16(current.Intelligence(), values, "intelligence"),
		mergeUint16(current.MaxHp(), values, "max_hp"),
		mergeUint16(current.MaxMp(), values, "max_mp"),
	)
}

// mergeUint16 returns the value at key coerced to uint16, or current if the
// key is absent. This is the crux of the fix: an absent key preserves the
// prior value instead of zeroing it.
func mergeUint16(current uint16, values map[string]interface{}, key string) uint16 {
	v, ok := values[key]
	if !ok {
		return current
	}
	return toUint16(v)
}

// toUint16 coerces any of the JSON-decode-compatible numeric types into
// uint16. Unknown types return 0.
func toUint16(v interface{}) uint16 {
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
