package character

import (
	"atlas-merchant/kafka/consumer"
	character2 "atlas-merchant/kafka/message/character"
	"atlas-merchant/shop"
	"context"

	consumer2 "github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer2.Config, decorators ...model.Decorator[consumer2.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer2.Config, decorators ...model.Decorator[consumer2.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer2.SetHeaderParsers(consumer2.SpanHeaderParser, consumer2.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			_, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLogout(db))))
			return err
		}
	}
}

func handleLogout(db *gorm.DB) func(logrus.FieldLogger, context.Context, character2.StatusEvent[character2.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type != character2.EventCharacterStatusTypeLogout {
			return
		}

		l.Debugf("Character [%d] logged out. Checking for active character shops.", e.CharacterId)

		p := shop.NewProcessor(l, ctx, db)
		shops, err := p.GetByCharacterId(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error retrieving shops for character [%d].", e.CharacterId)
			return
		}

		for _, s := range shops {
			if s.ShopType() == shop.CharacterShop && s.State() != shop.Closed {
				if err := p.CloseShopAndEmit(s.Id(), e.CharacterId, shop.CloseReasonDisconnect); err != nil {
					l.WithError(err).Errorf("Error closing character shop [%s] on disconnect.", s.Id())
				}
			}
		}
	}
}
