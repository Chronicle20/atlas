package shops

import (
	consumer2 "atlas-npc/kafka/consumer"
	shop2 "atlas-npc/kafka/message/shops"
	"atlas-npc/shops"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_shop_command")(shop2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(shop2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExitCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuyCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSellCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRechargeCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleEnterCommand(db *gorm.DB) message.Handler[shop2.Command[shop2.CommandShopEnterBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shop2.Command[shop2.CommandShopEnterBody]) {
		if e.Type != shop2.CommandShopEnter {
			return
		}
		err := shops.NewProcessor(l, ctx, db).EnterAndEmit(e.CharacterId, e.Body.NpcTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to process shop enter for character [%d] at NPC [%d].", e.CharacterId, e.Body.NpcTemplateId)
		}
	}
}

func handleExitCommand(db *gorm.DB) message.Handler[shop2.Command[shop2.CommandShopExitBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shop2.Command[shop2.CommandShopExitBody]) {
		if e.Type != shop2.CommandShopExit {
			return
		}
		err := shops.NewProcessor(l, ctx, db).ExitAndEmit(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to process shop exit for character [%d].", e.CharacterId)
		}
	}
}

func handleBuyCommand(db *gorm.DB) message.Handler[shop2.Command[shop2.CommandShopBuyBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shop2.Command[shop2.CommandShopBuyBody]) {
		if e.Type != shop2.CommandShopBuy {
			return
		}
		err := shops.NewProcessor(l, ctx, db).BuyAndEmit(e.CharacterId, e.Body.Slot, e.Body.ItemTemplateId, e.Body.Quantity, e.Body.DiscountPrice)
		if err != nil {
			l.WithError(err).Errorf("Unable to process buy for character [%d], item [%d], quantity [%d].", e.CharacterId, e.Body.ItemTemplateId, e.Body.Quantity)
		}
	}
}

func handleSellCommand(db *gorm.DB) message.Handler[shop2.Command[shop2.CommandShopSellBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shop2.Command[shop2.CommandShopSellBody]) {
		if e.Type != shop2.CommandShopSell {
			return
		}
		err := shops.NewProcessor(l, ctx, db).SellAndEmit(e.CharacterId, e.Body.Slot, e.Body.ItemTemplateId, e.Body.Quantity)
		if err != nil {
			l.WithError(err).Errorf("Unable to process sell for character [%d], item [%d], quantity [%d].", e.CharacterId, e.Body.ItemTemplateId, e.Body.Quantity)
		}
	}
}

func handleRechargeCommand(db *gorm.DB) message.Handler[shop2.Command[shop2.CommandShopRechargeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shop2.Command[shop2.CommandShopRechargeBody]) {
		if e.Type != shop2.CommandShopRecharge {
			return
		}
		err := shops.NewProcessor(l, ctx, db).RechargeAndEmit(e.CharacterId, e.Body.Slot)
		if err != nil {
			l.WithError(err).Errorf("Unable to process recharge for character [%d], slot [%d].", e.CharacterId, e.Body.Slot)
		}
	}
}
