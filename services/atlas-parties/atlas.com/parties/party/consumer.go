package party

import (
	consumer2 "atlas-parties/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const consumerCommand = "party_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func CreateCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleCreate))
}

func handleCreate(l logrus.FieldLogger, ctx context.Context, c commandEvent[createCommandBody]) {
	if c.Type != CommandPartyCreate {
		return
	}
	_, err := Create(l)(ctx)(c.Body.LeaderId)
	if err != nil {
		l.WithError(err).Errorf("Unable to create party for leader [%d].", c.Body.LeaderId)
	}
}

func JoinCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleJoin))
}

func handleJoin(l logrus.FieldLogger, ctx context.Context, c commandEvent[joinCommandBody]) {
	if c.Type != CommandPartyJoin {
		return
	}
	_, err := Join(l)(ctx)(c.Body.PartyId, c.Body.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join party [%d].", c.Body.CharacterId, c.Body.PartyId)
	}
}

func LeaveCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLeave))
}

func handleLeave(l logrus.FieldLogger, ctx context.Context, c commandEvent[leaveCommandBody]) {
	if c.Type != CommandPartyLeave {
		return
	}

	if c.Body.Force {
		_, err := Expel(l)(ctx)(c.Body.PartyId, c.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to expel [%d] from party [%d].", c.Body.CharacterId, c.Body.PartyId)
			return
		}
	} else {
		_, err := Leave(l)(ctx)(c.Body.PartyId, c.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to leave party [%d].", c.Body.PartyId)
			return
		}
	}
}

func ChangeLeaderCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleChangeLeader))
}

func handleChangeLeader(l logrus.FieldLogger, ctx context.Context, c commandEvent[changeLeaderBody]) {
	if c.Type != CommandPartyChangeLeader {
		return
	}
	_, err := ChangeLeader(l)(ctx)(c.Body.PartyId, c.Body.LeaderId)
	if err != nil {
		l.WithError(err).Errorf("Unable to establish [%d] as leader of party [%d].", c.Body.LeaderId, c.Body.PartyId)
	}
}
