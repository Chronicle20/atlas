package character

import (
	"atlas-guilds/guild"
	"atlas-guilds/guild/member"
	consumer2 "atlas-guilds/kafka/consumer"
	character2 "atlas-guilds/kafka/message/character"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			if err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterNameChanged(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventDeletedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventDeletedBody]) {
		if e.Type != character2.EventCharacterStatusTypeDeleted {
			return
		}

		p := guild.NewProcessor(l, ctx, db)
		g, err := p.GetByMemberId(e.CharacterId)
		if err != nil {
			// Character not in a guild, nothing to clean up
			return
		}

		err = p.LeaveAndEmit(g.Id(), e.CharacterId, true, uuid.New())
		if err != nil {
			l.WithError(err).Errorf("Unable to remove deleted character [%d] from guild [%d].", e.CharacterId, g.Id())
		}
	}
}

func handleStatusEventLogin(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
		if e.Type != character2.EventCharacterStatusTypeLogin {
			return
		}

		err := guild.NewProcessor(l, ctx, db).UpdateMemberOnlineAndEmit(e.CharacterId, true, uuid.New())
		if err != nil {
			l.WithError(err).Errorf("Unable to process login for character [%d].", e.CharacterId)
		}
	}
}

func handleStatusEventLogout(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type != character2.EventCharacterStatusTypeLogout {
			return
		}

		err := guild.NewProcessor(l, ctx, db).UpdateMemberOnlineAndEmit(e.CharacterId, false, uuid.New())
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", e.CharacterId)
		}
	}
}

func handleCharacterNameChanged(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventNameChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventNameChangedBody]) {
		if e.Type != character2.EventCharacterStatusTypeNameChanged {
			return
		}

		err := member.NewProcessor(l, ctx, db).UpdateName(e.CharacterId, e.Body.NewName)
		if err != nil {
			l.WithError(err).Errorf("Unable to update guild roster name for character [%d] from [%s] to [%s].", e.CharacterId, e.Body.OldName, e.Body.NewName)
			return
		}
		t := tenant.MustFromContext(ctx)
		l.WithFields(logrus.Fields{
			"tenant":      t.Id(),
			"characterId": e.CharacterId,
			"oldName":     e.Body.OldName,
			"newName":     e.Body.NewName,
		}).Infof("Updated guild roster name for character [%d].", e.CharacterId)
	}
}
