package character

import (
	"atlas-maps/character/warp"
	characterKafka "atlas-maps/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// changeMapFromCommand builds the destination field from a CHANGE_MAP command
// body and funnels it through the single shared warp method.
func changeMapFromCommand(wp warp.Processor) func(c characterKafka.Command[characterKafka.ChangeMapBody]) error {
	return func(c characterKafka.Command[characterKafka.ChangeMapBody]) error {
		dest := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		return wp.ChangeMap(c.TransactionId, c.CharacterId, c.WorldId, dest, c.Body.PortalId)
	}
}

// handleChangeMapFunc consumes CHANGE_MAP commands from the
// COMMAND_TOPIC_CHARACTER topic and funnels them through the single shared
// warp.Processor.ChangeMap method, which persists the new field via
// location.Set, emits the canonical MAP_CHANGED status event, and updates
// atlas-maps' own per-map registries via _map.Processor.TransitionMapAndEmit.
//
// This consumer was migrated from atlas-character as part of the
// forced-return-on-exit work (task-055): atlas-maps now owns location state,
// so the warp's authoritative point of truth lives here. task-087 factored the
// body into warp.Processor so the REST location path and this command path
// share one implementation.
func handleChangeMapFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
		if c.Type != characterKafka.CommandChangeMap {
			return
		}
		wp := warp.NewProcessor(l, ctx, db)
		if err := changeMapFromCommand(wp)(c); err != nil {
			l.WithError(err).Errorf("CHANGE_MAP: warp failed for character [%d].", c.CharacterId)
		}
	}
}
