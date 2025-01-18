package session

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	tenant "github.com/Chronicle20/atlas-tenant"
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
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
		l.Debugf("Received session status event. sessionId [%s] accountId [%d] characterId [%d] worldId [%d] channelId [%d] issuer [%s] type [%s].", event.SessionId.String(), event.AccountId, event.CharacterId, event.WorldId, event.ChannelId, event.Issuer, event.Type)
		if event.Issuer != EventSessionStatusIssuerChannel {
			return
		}

		t := tenant.MustFromContext(ctx)
		if event.Type == EventSessionStatusTypeCreated {
			cs, err := GetRegistry().Get(t, event.CharacterId)
			if err != nil || cs.State() == StateLoggedOut {
				l.Debugf("Processing a session status event of [%s] which will trigger a login.", event.Type)
				err = GetRegistry().Add(t, event.CharacterId, event.WorldId, event.ChannelId, StateLoggedIn)
				if err != nil {
					err = character.Login(l)(db)(ctx)(event.CharacterId)(event.WorldId)(event.ChannelId)
					if err != nil {
						l.WithError(err).Errorf("Unable to login character [%d] as a result of session [%s] being created.", event.CharacterId, event.SessionId.String())
					}
				}
			} else if cs.State() == StateTransition {
				l.Debugf("Processing a session status event of [%s] which will trigger a change channel.", event.Type)
				err = GetRegistry().Set(t, event.CharacterId, event.WorldId, event.ChannelId, StateLoggedIn)
				if err != nil {
					err = character.ChangeChannel(l)(db)(ctx)(event.CharacterId)(event.WorldId)(event.ChannelId)(cs.ChannelId())
					if err != nil {
						l.WithError(err).Errorf("Unable to change character [%d] channel as a result of session [%s] being created.", event.CharacterId, event.SessionId.String())
					}
				}
			}
			return
		}
		if event.Type == EventSessionStatusTypeDestroyed {
			cs, err := GetRegistry().Get(t, event.CharacterId)
			if err != nil {
				l.Debugf("Processing a session status event of [%s]. Session already destroyed.", event.Type)
				return
			}
			if cs.State() == StateLoggedIn {
				l.Debugf("Processing a session status event of [%s] which will trigger a transition state. Either it will be culled (logout), or updated (change channel) later.", event.Type)
				_ = GetRegistry().Set(t, cs.CharacterId(), cs.WorldId(), cs.ChannelId(), StateTransition)
			}
			return
		}
	}
}
