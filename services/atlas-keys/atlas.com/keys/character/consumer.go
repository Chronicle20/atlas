package character

import (
	consumer2 "atlas-keys/kafka/consumer"
	"atlas-keys/key"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerCharacterCreated = "character_created"
)

func CreatedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCharacterCreated)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func CreatedStatusEventRegister(db *gorm.DB) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(db)))
	}
}

func handleCreatedStatusEvent(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
		err := key.CreateDefault(l)(ctx)(db)(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to create default keymapping for character %d.", e.CharacterId)
		}
	}
}
