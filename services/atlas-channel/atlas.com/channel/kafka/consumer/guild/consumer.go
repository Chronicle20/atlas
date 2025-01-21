package guild

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const consumerCommand = "guild_command"
const consumerStatusEvent = "guild_status_event"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func RequestNameRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleRequestNameCommand(sc, wp)))
	}
}

func handleRequestNameCommand(sc server.Model, wp writer.Producer) message.Handler[command[requestNameBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestNameBody]) {
		if c.Type != CommandTypeRequestName {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.Body.WorldId, c.Body.ChannelId) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, requestGuildName(l)(ctx)(wp))
	}
}

func requestGuildName(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			guildOperationFunc := session.Announce(l)(ctx)(wp)(writer.GuildOperation)
			return func(s session.Model) error {
				err := guildOperationFunc(s, writer.RequestGuildNameBody(l))
				if err != nil {
					l.Debugf("Unable to request character [%d] input guild name.", s.CharacterId())
					return err
				}
				return nil
			}
		}
	}
}

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvStatusEventTopic)(groupId)
	}
}

func RequestAgreementStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventRequestAgreementBody]) {
			if e.Type != StatusEventTypeRequestAgreement {
				return
			}

			t := sc.Tenant()
			if !t.Is(tenant.MustFromContext(ctx)) {
				return
			}
			if sc.WorldId() != e.WorldId {
				return
			}

			p, err := party.GetByMemberId(l)(ctx)(e.Body.ActorId)
			if err != nil {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(writer.GuildOperationCreateError))
				return
			}

			leaderName := ""
			for _, m := range p.Members() {
				if p.LeaderId() == m.Id() {
					leaderName = m.Name()
					break
				}
			}

			for _, m := range p.Members() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(m.Id(), requestGuildNameAgreement(l)(ctx)(wp)(p.Id(), leaderName, e.Body.ProposedName))
			}
		}))
	}
}

func requestGuildNameAgreement(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
			guildOperationFunc := session.Announce(l)(ctx)(wp)(writer.GuildOperation)
			return func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := guildOperationFunc(s, writer.GuildRequestAgreement(l)(partyId, leaderName, guildName))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				}
			}
		}
	}
}

func ErrorStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventErrorBody]) {
			if e.Type != StatusEventTypeError {
				return
			}

			t := sc.Tenant()
			if !t.Is(tenant.MustFromContext(ctx)) {
				return
			}
			if sc.WorldId() != e.WorldId {
				return
			}

			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(e.Body.Error))
		}))
	}
}

func announceGuildError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
			guildOperationFunc := session.Announce(l)(ctx)(wp)(writer.GuildOperation)
			return func(errCode string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := guildOperationFunc(s, writer.GuildErrorBody(l)(errCode))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				}
			}
		}
	}
}
