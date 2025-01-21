package guild

import (
	consumer2 "atlas-channel/kafka/consumer"
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
