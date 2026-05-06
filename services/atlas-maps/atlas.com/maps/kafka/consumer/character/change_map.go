package character

import (
	"atlas-maps/character/location"
	"atlas-maps/kafka/message"
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// handleChangeMapFunc consumes CHANGE_MAP commands from the
// COMMAND_TOPIC_CHARACTER topic, persists the new field via
// location.Set, emits the canonical MAP_CHANGED status event, and updates
// atlas-maps' own per-map registries via _map.Processor.TransitionMapAndEmit.
//
// This consumer was migrated from atlas-character as part of the
// forced-return-on-exit work (task-055): atlas-maps now owns location state,
// so the warp's authoritative point of truth lives here rather than in
// atlas-character.
func handleChangeMapFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
		if c.Type != characterKafka.CommandChangeMap {
			return
		}

		newField := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()

		lp := location.NewProcessor(l, ctx, db)
		old, err := lp.GetById(c.CharacterId)
		oldField := newField
		if err == nil {
			oldField = old.Field()
		}

		if _, err := lp.Set(c.CharacterId, newField); err != nil {
			l.WithError(err).Errorf("CHANGE_MAP: location.Set failed for character [%d].", c.CharacterId)
			return
		}

		pp := producer.ProviderImpl(l)(ctx)
		if err := message.Emit(pp)(func(buf *message.Buffer) error {
			return buf.Put(characterKafka.EnvEventTopicCharacterStatus,
				producer.MapChangedStatusProvider(c.TransactionId, c.CharacterId, c.WorldId, oldField, newField, c.Body.PortalId))
		}); err != nil {
			l.WithError(err).Errorf("CHANGE_MAP: failed to emit MAP_CHANGED status for character [%d].", c.CharacterId)
		}

		mp := _map.NewProcessor(l, ctx, pp, db)
		if err := mp.TransitionMapAndEmit(c.TransactionId, newField, c.CharacterId, oldField); err != nil {
			l.WithError(err).Warnf("CHANGE_MAP: TransitionMapAndEmit failed for character [%d].", c.CharacterId)
		}
	}
}
