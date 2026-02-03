package npc

import (
	npc2 "atlas-channel/kafka/message/npc"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func StartConversationCommandProvider(f field.Model, npcId uint32, characterId uint32, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.Command[npc2.StartConversationCommandBody]{
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        npc2.CommandTypeStartConversation,
		Body: npc2.StartConversationCommandBody{
			WorldId:   f.WorldId(),
			ChannelId: f.ChannelId(),
			MapId:     f.MapId(),
			Instance:  f.Instance(),
			AccountId: accountId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ContinueConversationCommandProvider(characterId uint32, action byte, lastMessageType byte, selection int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.Command[npc2.ContinueConversationCommandBody]{
		NpcId:       0, // TODO
		CharacterId: characterId,
		Type:        npc2.CommandTypeContinueConversation,
		Body: npc2.ContinueConversationCommandBody{
			Action:          action,
			LastMessageType: lastMessageType,
			Selection:       selection,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DisposeConversationCommandProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.Command[npc2.EndConversationCommandBody]{
		NpcId:       0, // TODO
		CharacterId: characterId,
		Type:        npc2.CommandTypeEndConversation,
		Body:        npc2.EndConversationCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
