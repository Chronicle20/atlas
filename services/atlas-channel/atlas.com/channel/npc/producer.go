package npc

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func startConversationCommandProvider(worldId byte, channelId byte, mapId uint32, npcId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[startConversationCommandBody]{
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        CommandTypeStartConversation,
		Body: startConversationCommandBody{
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func continueConversationCommandProvider(characterId uint32, action byte, lastMessageType byte, selection int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[continueConversationCommandBody]{
		NpcId:       0, // TODO
		CharacterId: characterId,
		Type:        CommandTypeContinueConversation,
		Body: continueConversationCommandBody{
			Action:          action,
			LastMessageType: lastMessageType,
			Selection:       selection,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func disposeConversationCommandProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[endConversationCommandBody]{
		NpcId:       0, // TODO
		CharacterId: characterId,
		Type:        CommandTypeEndConversation,
		Body:        endConversationCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
