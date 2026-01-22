package quest

import (
	quest2 "atlas-channel/kafka/message/quest"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func StartConversationCommandProvider(m _map.Model, questId uint32, npcId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.Command[quest2.StartQuestConversationCommandBody]{
		QuestId:     questId,
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        quest2.CommandTypeStartQuestConversation,
		Body: quest2.StartQuestConversationCommandBody{
			WorldId:   m.WorldId(),
			ChannelId: m.ChannelId(),
			MapId:     m.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StartQuestCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, questId uint32, npcId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.StartQuestCommandBody]{
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		MapId:       uint32(mapId),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeStart,
		Body: quest2.StartQuestCommandBody{
			QuestId: questId,
			NpcId:   npcId,
			Force:   force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CompleteQuestCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.CompleteQuestCommandBody]{
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		MapId:       uint32(mapId),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeComplete,
		Body: quest2.CompleteQuestCommandBody{
			QuestId:   questId,
			NpcId:     npcId,
			Selection: selection,
			Force:     force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ForfeitQuestCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.ForfeitQuestCommandBody]{
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		MapId:       uint32(mapId),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeForfeit,
		Body: quest2.ForfeitQuestCommandBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RestoreItemCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, questId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.RestoreItemCommandBody]{
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		MapId:       uint32(mapId),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeRestoreItem,
		Body: quest2.RestoreItemCommandBody{
			QuestId: questId,
			ItemId:  itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
