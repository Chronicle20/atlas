package _map

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/monster"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicMapStatus)(groupId)
	}
}

func StatusEventCharacterEnterRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter))
}

func StatusEventCharacterExitRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))
}

func handleStatusEventCharacterEnter(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterEnter]) {
	if event.Type != EventTopicMapStatusTypeCharacterEnter {
		return
	}

	provider := monster.NotControlledInMapProvider(ctx)(event.WorldId, event.ChannelId, event.MapId)
	_ = model.ForEachSlice(provider, monster.FindNextController(l)(ctx), model.ParallelExecute())
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterExit]) {
	if event.Type != EventTopicMapStatusTypeCharacterExit {
		return
	}

	provider := monster.ControlledByCharacterInMapProvider(ctx)(event.WorldId, event.ChannelId, event.MapId, event.Body.CharacterId)
	_ = model.ForEachSlice(provider, monster.StopControl(l)(ctx), model.ParallelExecute())
	_ = model.ForEachSlice(provider, monster.FindNextController(l)(ctx))
}
