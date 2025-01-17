package cashshop

import (
	consumer2 "atlas-buddies/kafka/consumer"
	"atlas-buddies/list"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerStatusEvent = "cash_shop_status"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCashShopStatus)(groupId)
	}
}

func CharacterEnterStatusRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCashShopStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(db)))
	}
}

func handleStatusEventCharacterEnter(db *gorm.DB) message.Handler[statusEvent[characterMovementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterMovementBody]) {
		if e.Type != EventCashShopStatusTypeCharacterEnter {
			return
		}
		_ = list.UpdateShopStatus(l)(ctx)(db)(e.Body.CharacterId, e.WorldId, true)
	}
}

func CharacterExitStatusRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCashShopStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(db)))
	}
}

func handleStatusEventCharacterExit(db *gorm.DB) message.Handler[statusEvent[characterMovementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterMovementBody]) {
		if e.Type != EventCashShopStatusTypeCharacterExit {
			return
		}
		_ = list.UpdateShopStatus(l)(ctx)(db)(e.Body.CharacterId, e.WorldId, false)
	}
}
