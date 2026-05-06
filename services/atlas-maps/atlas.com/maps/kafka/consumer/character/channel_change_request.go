package character

import (
	"atlas-maps/character/location"
	"atlas-maps/kafka/message"
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// handleChannelChangeRequestFunc consumes CHANNEL_CHANGE_REQUEST commands
// from atlas-channel and resolves the target field through the same
// forced-return policy used on LOGOUT/LOGIN. The resolved field is persisted
// via location.Set, then atlas-maps emits the canonical CHANNEL_CHANGED
// status event and updates its own per-channel registries via
// _map.Processor.TransitionChannelAndEmit.
func handleChannelChangeRequestFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c characterKafka.ChannelChangeRequestCommand) {
	return func(l logrus.FieldLogger, ctx context.Context, c characterKafka.ChannelChangeRequestCommand) {
		l.Debugf("Received CHANNEL_CHANGE_REQUEST for character [%d]: worldId [%d] oldChannelId [%d] targetChannelId [%d].", c.CharacterId, c.WorldId, c.OldChannelId, c.TargetChannelId)

		lp := location.NewProcessor(l, ctx, db)
		cur, err := lp.GetById(c.CharacterId)
		if err != nil {
			l.WithError(err).Warnf("CHANNEL_CHANGE_REQUEST: location.GetById failed for character [%d]; skipping.", c.CharacterId)
			return
		}

		target := field.NewBuilder(cur.WorldId(), c.TargetChannelId, cur.MapId()).SetInstance(cur.Instance()).Build()

		resolved, reason, err := lp.Resolve(target)
		if err != nil {
			l.WithError(err).Warnf("CHANNEL_CHANGE_REQUEST: location.Resolve failed for character [%d]; staying put on target channel.", c.CharacterId)
			resolved = target
			reason = location.ReasonStayPut
		}

		if _, err := lp.Set(c.CharacterId, resolved); err != nil {
			l.WithError(err).Errorf("CHANNEL_CHANGE_REQUEST: location.Set failed for character [%d]; aborting.", c.CharacterId)
			return
		}

		if reason != location.ReasonStayPut {
			l.WithFields(logrus.Fields{
				"character_id":      c.CharacterId,
				"target_channel":    c.TargetChannelId,
				"resolved_map_id":   resolved.MapId(),
				"resolution_reason": string(reason),
			}).Info("forced-return resolution on CHANNEL_CHANGE_REQUEST")
		}

		newField := resolved
		transactionId := uuid.New()

		pp := producer.ProviderImpl(l)(ctx)
		if err := message.Emit(pp)(func(buf *message.Buffer) error {
			return buf.Put(characterKafka.EnvEventTopicCharacterStatus, producer.ChannelChangedStatusProvider(transactionId, c.CharacterId, c.WorldId, c.OldChannelId, newField))
		}); err != nil {
			l.WithError(err).Errorf("CHANNEL_CHANGE_REQUEST: failed to emit CHANNEL_CHANGED status for character [%d].", c.CharacterId)
		}

		mp := _map.NewProcessor(l, ctx, pp, db)
		if err := mp.TransitionChannelAndEmit(transactionId, newField, c.OldChannelId, c.CharacterId); err != nil {
			l.WithError(err).Warnf("CHANNEL_CHANGE_REQUEST: TransitionChannelAndEmit failed for character [%d].", c.CharacterId)
		}
	}
}
