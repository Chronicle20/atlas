package food

import (
	consumer2 "atlas-mounts/kafka/consumer"
	mountmessage "atlas-mounts/kafka/message"
	foodmsg "atlas-mounts/kafka/message/food"
	"atlas-mounts/mount"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// Function seam. Production wires this to the real processor, emitting the FEED
// status event through the buffer; tests override it with a fake to exercise the
// handler's argument threading without Kafka or a database.
var applyFeed = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, worldId world.Id, characterId uint32, healMax int) error {
	return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		p := mount.NewProcessor(l, ctx, tx)
		return mountmessage.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *mountmessage.Buffer) error {
			return p.ApplyFeedAndEmit(mb)(worldId, characterId, healMax)
		})
	})
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("taming_mob_food_event")(foodmsg.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(foodmsg.EnvEventTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleTamingMobFood(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleTamingMobFood(db *gorm.DB) message.Handler[foodmsg.Event] {
	return func(l logrus.FieldLogger, ctx context.Context, e foodmsg.Event) {
		l.Debugf("Taming mob fed for character [%d]: item [%d] tirednessHeal [%d].", e.CharacterId, e.ItemId, e.TirednessHeal)
		if err := applyFeed(l, ctx, db, e.WorldId, e.CharacterId, int(e.TirednessHeal)); err != nil {
			l.WithError(err).Errorf("Unable to apply mount feed for character [%d].", e.CharacterId)
		}
	}
}
