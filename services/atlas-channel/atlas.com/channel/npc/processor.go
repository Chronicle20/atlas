package npc

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, f model.Operator[Model]) error {
	return func(ctx context.Context) func(mapId _map.Id, f model.Operator[Model]) error {
		return func(mapId _map.Id, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(mapId), f, model.ParallelExecute())
		}
	}
}

func InMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId _map.Id) model.Provider[[]Model] {
		return func(mapId _map.Id) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestNPCsInMap(mapId), Extract, model.Filters[Model]())
		}
	}
}

func InMapByObjectIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
		return func(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestNPCsInMapByObjectId(mapId, objectId), Extract, model.Filters[Model]())
		}
	}
}

func GetInMapByObjectId(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, objectId uint32) (Model, error) {
	return func(ctx context.Context) func(mapId _map.Id, objectId uint32) (Model, error) {
		return func(mapId _map.Id, objectId uint32) (Model, error) {
			p := InMapByObjectIdModelProvider(l)(ctx)(mapId, objectId)
			return model.First[Model](p, model.Filters[Model]())
		}
	}
}

func StartConversation(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, npcId uint32, characterId uint32) error {
	return func(ctx context.Context) func(m _map.Model, npcId uint32, characterId uint32) error {
		return func(m _map.Model, npcId uint32, characterId uint32) error {
			l.Debugf("Starting NPC [%d] conversation for character [%d].", characterId, npcId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(startConversationCommandProvider(m, npcId, characterId))
		}
	}
}

func ContinueConversation(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, action byte, lastMessageType byte, selection int32) error {
	return func(ctx context.Context) func(characterId uint32, action byte, lastMessageType byte, selection int32) error {
		return func(characterId uint32, action byte, lastMessageType byte, selection int32) error {
			l.Debugf("Continuing NPC conversation for character [%d]. action [%d], lastMessageType [%d], selection [%d].", characterId, action, lastMessageType, selection)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(continueConversationCommandProvider(characterId, action, lastMessageType, selection))
		}
	}
}

func DisposeConversation(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			l.Debugf("Ending NPC conversation for character [%d].", characterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(disposeConversationCommandProvider(characterId))
		}
	}
}
