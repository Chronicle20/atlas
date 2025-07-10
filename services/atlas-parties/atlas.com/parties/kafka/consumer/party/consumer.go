package party

import (
	consumer2 "atlas-parties/kafka/consumer"
	"atlas-parties/party"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreate)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeave)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeLeader)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestInvite)))
	}
}

func handleCreate(l logrus.FieldLogger, ctx context.Context, c commandEvent[createCommandBody]) {
	if c.Type != CommandPartyCreate {
		return
	}
	_, err := party.NewProcessor(l, ctx).CreateAndEmit(c.ActorId)
	if err != nil {
		l.WithError(err).Errorf("Unable to create party for leader [%d].", c.ActorId)
	}
}

func handleJoin(l logrus.FieldLogger, ctx context.Context, c commandEvent[joinCommandBody]) {
	if c.Type != CommandPartyJoin {
		return
	}
	_, err := party.NewProcessor(l, ctx).JoinAndEmit(c.Body.PartyId, c.ActorId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join party [%d].", c.ActorId, c.Body.PartyId)
	}
}

func handleLeave(l logrus.FieldLogger, ctx context.Context, c commandEvent[leaveCommandBody]) {
	if c.Type != CommandPartyLeave {
		return
	}

	if c.Body.Force {
		_, err := party.NewProcessor(l, ctx).ExpelAndEmit(c.ActorId, c.Body.PartyId, c.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to expel [%d] from party [%d].", c.ActorId, c.Body.PartyId)
			return
		}
	} else {
		_, err := party.NewProcessor(l, ctx).LeaveAndEmit(c.Body.PartyId, c.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to leave party [%d].", c.Body.PartyId)
			return
		}
	}
}

func handleChangeLeader(l logrus.FieldLogger, ctx context.Context, c commandEvent[changeLeaderBody]) {
	if c.Type != CommandPartyChangeLeader {
		return
	}
	_, err := party.NewProcessor(l, ctx).ChangeLeaderAndEmit(c.ActorId, c.Body.PartyId, c.Body.LeaderId)
	if err != nil {
		l.WithError(err).Errorf("Unable to establish [%d] as leader of party [%d].", c.Body.LeaderId, c.Body.PartyId)
	}
}

func handleRequestInvite(l logrus.FieldLogger, ctx context.Context, c commandEvent[requestInviteBody]) {
	if c.Type != CommandPartyRequestInvite {
		return
	}
	err := party.NewProcessor(l, ctx).RequestInviteAndEmit(c.ActorId, c.Body.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to invite [%d] to party.", c.Body.CharacterId)
	}
}
