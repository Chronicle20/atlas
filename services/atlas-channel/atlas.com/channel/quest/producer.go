package quest

import (
	quest2 "atlas-channel/kafka/message/quest"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func StartConversationCommandProvider(f field.Model, questId uint32, npcId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.Command[quest2.StartQuestConversationCommandBody]{
		QuestId:     questId,
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        quest2.CommandTypeStartQuestConversation,
		Body: quest2.StartQuestConversationCommandBody{
			WorldId:   f.WorldId(),
			ChannelId: f.ChannelId(),
			MapId:     f.MapId(),
			Instance:  f.Instance(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StartQuestCommandProvider(f field.Model, characterId uint32, questId uint32, npcId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.StartQuestCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
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

func CompleteQuestCommandProvider(f field.Model, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.CompleteQuestCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
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

func ForfeitQuestCommandProvider(f field.Model, characterId uint32, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.ForfeitQuestCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeForfeit,
		Body: quest2.ForfeitQuestCommandBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RestoreItemCommandProvider(f field.Model, characterId uint32, questId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.QuestCommand[quest2.RestoreItemCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        quest2.QuestCommandTypeRestoreItem,
		Body: quest2.RestoreItemCommandBody{
			QuestId: questId,
			ItemId:  itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
