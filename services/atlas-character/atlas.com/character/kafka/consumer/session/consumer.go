package session

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	session2 "atlas-character/kafka/message/session"
	"atlas-character/session"
	"atlas-character/session/history"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(session2.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(session2.EnvEventTopicSessionStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent(db))))
		}
	}
}

func handleStatusEvent(db *gorm.DB) message.Handler[session2.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent) {
		l.Debugf("Received session status event. sessionId [%s] accountId [%d] characterId [%d] worldId [%d] channelId [%d] issuer [%s] type [%s].", e.SessionId.String(), e.AccountId, e.CharacterId, e.WorldId, e.ChannelId, e.Issuer, e.Type)
		if e.Issuer != session2.EventSessionStatusIssuerChannel {
			return
		}

		hp := history.NewProcessor(l, ctx, db)

		ch := channel.NewModel(e.WorldId, e.ChannelId)
		if e.Type == session2.EventSessionStatusTypeCreated {
			cs, err := session.GetRegistry().Get(ctx, e.CharacterId)
			if err != nil || cs.State() == session.StateLoggedOut {
				l.Debugf("Processing a session status event of [%s] which will trigger a login.", e.Type)
				err = session.GetRegistry().Add(ctx, e.CharacterId, ch, session.StateLoggedIn)
				if err != nil {
					l.WithError(err).Errorf("Character [%d] already logged in. Eating event.", e.CharacterId)
					return
				}

				// Start session history tracking
				_, err = hp.StartSession(e.CharacterId, ch)
				if err != nil {
					l.WithError(err).Warnf("Failed to start session history for character [%d].", e.CharacterId)
				}

				err = character.NewProcessor(l, ctx, db).LoginAndEmit(uuid.New(), e.CharacterId, ch)
				if err != nil {
					l.WithError(err).Errorf("Unable to login character [%d] as a result of session [%s] being created.", e.CharacterId, e.SessionId.String())
				}
				return
			} else if cs.State() == session.StateTransition {
				l.Debugf("Processing a session status event of [%s] which will trigger a change channel.", e.Type)
				err = session.GetRegistry().Set(ctx, e.CharacterId, ch, session.StateLoggedIn)

				// End previous session and start new one for channel change
				_ = hp.EndSession(e.CharacterId)
				_, err = hp.StartSession(e.CharacterId, ch)
				if err != nil {
					l.WithError(err).Warnf("Failed to start session history for character [%d] on channel change.", e.CharacterId)
				}

				err = character.NewProcessor(l, ctx, db).ChangeChannelAndEmit(uuid.New(), e.CharacterId, ch, cs.ChannelId())
				if err != nil {
					l.WithError(err).Errorf("Unable to change character [%d] channel as a result of session [%s] being created.", e.CharacterId, e.SessionId.String())
				}
			}
			return
		}
		if e.Type == session2.EventSessionStatusTypeDestroyed {
			cs, err := session.GetRegistry().Get(ctx, e.CharacterId)
			if err != nil {
				l.Debugf("Processing a session status event of [%s]. Session already destroyed.", e.Type)
				return
			}
			if cs.State() == session.StateLoggedIn {
				l.Debugf("Processing a session status event of [%s] which will trigger a transition state. Either it will be culled (logout), or updated (change channel) later.", e.Type)
				och := channel.NewModel(cs.WorldId(), cs.ChannelId())
				_ = session.GetRegistry().Set(ctx, cs.CharacterId(), och, session.StateTransition)
			}
			return
		}
	}
}
