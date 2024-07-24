package session

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicSessionStatus)(groupId)
	}
}

func StatusEventRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicSessionStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent(db)))
}

func handleStatusEvent(db *gorm.DB) message.Handler[statusEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent) {
		l.Debugf("Received session status event. sessionId [%d] accountId [%d] characterId [%d] worldId [%d] channelId [%d] issuer [%s] type [%s].", event.SessionId, event.AccountId, event.CharacterId, event.WorldId, event.ChannelId, event.Issuer, event.Type)
		if event.Issuer != EventSessionStatusIssuerChannel {
			return
		}

		if event.Type == EventSessionStatusTypeCreated {
			character.Login(l, db, span, event.Tenant)(event.CharacterId, event.WorldId, event.ChannelId)
			return
		}
		if event.Type == EventSessionStatusTypeDestroyed {
			character.Logout(l, db, span, event.Tenant)(event.CharacterId, event.WorldId, event.ChannelId)
			return
		}
	}
}
