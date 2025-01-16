package list

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
		if c.Type != CommandTypeCreate {
			return
		}
		_, err := list.Create(l)(ctx)(db)(c.CharacterId, c.Body.Capacity)
		if err != nil {
			l.WithError(err).Errorf("Error creating buddy list for character [%d].", c.CharacterId)
		}
	}
}

func RequestAddCommandRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyAddCommand(db)))
	}
}

func handleRequestBuddyAddCommand(db *gorm.DB) message.Handler[command[requestAddBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestAddBuddyCommandBody]) {
		if c.Type != CommandTypeRequestAdd {
			return
		}
		err := list.RequestAdd(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.CharacterId, c.Body.CharacterName, c.Body.Group)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to add [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}

func RequestDeleteCommandRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyDeleteCommand(db)))
	}
}

func handleRequestBuddyDeleteCommand(db *gorm.DB) message.Handler[command[requestDeleteBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDeleteBuddyCommandBody]) {
		if c.Type != CommandTypeRequestDelete {
			return
		}
		err := list.RequestDelete(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to delete [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}
