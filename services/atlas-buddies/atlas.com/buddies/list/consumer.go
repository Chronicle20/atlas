package list

import (
	consumer2 "atlas-buddies/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerCommand = "buddy_list_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func CreateCommandRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleCreateBuddyListCommand(db)))
	}
}

func handleCreateBuddyListCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
		_, err := Create(l)(ctx)(db)(c.CharacterId, c.Body.Capacity)
		if err != nil {
			l.WithError(err).Errorf("Error creating buddy list for character [%d].", c.CharacterId)
		}
	}
}
