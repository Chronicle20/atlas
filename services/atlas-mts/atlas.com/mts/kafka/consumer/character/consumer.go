package character

import (
	consumer2 "atlas-mts/kafka/consumer"
	"atlas-mts/kafka/message/character"
	"atlas-mts/listing"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers registers the character status event consumer, mirroring the
// shape of the custody/mts command consumers. atlas-mts has no push path for
// listing browsing (clients re-query rather than receiving pushes), so this
// consumer only keeps stored state (listing.seller_name) current — it emits
// nothing.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mts_character_status")(character.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

// InitHandlers wires the NAME_CHANGED handler onto the character status topic.
// Every other status type on this topic is filtered out by the handler's type
// guard.
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(character.EnvStatusEventTopic)()
			if err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterNameChanged(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleCharacterNameChanged keeps listing.seller_name current for the renamed
// character. seller_name is the seller's current display identity, not a
// point-in-time record of who they were at sale time, so EVERY listing row for
// that seller is renamed regardless of State (active, sold, cancelled,
// expired). The rename is a single UPDATE keyed on seller_id; a redelivery of
// the same event re-applies the same NewName, which is a no-op past the first
// delivery. No event is emitted — MTS browsing is request/response (clients
// re-query listings), so there is no push path to feed.
func handleCharacterNameChanged(db *gorm.DB) message.Handler[character.StatusEvent[character.StatusEventNameChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.StatusEventNameChangedBody]) {
		if e.Type != character.StatusEventTypeNameChanged {
			return
		}
		if _, err := listing.NewProcessor(l, ctx, db).RenameSeller(e.CharacterId, e.Body.NewName); err != nil {
			l.WithError(err).Errorf("Unable to rename seller [%d]'s listings to [%s].", e.CharacterId, e.Body.NewName)
		}
	}
}
