package npc

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enableActionsProvider(worldId world.Id, channelId channel.Id, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.StatusEvent[npc2.StatusEventStatChangedBody]{
		CharacterId: characterId,
		Type:        npc2.EventCharacterStatusTypeStatChanged,
		WorldId:     byte(worldId),
		Body: npc2.StatusEventStatChangedBody{
			ChannelId:       byte(channelId),
			ExclRequestSent: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func simpleConversationProvider(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, messageType string, speaker string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandSimpleBody]{
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		CharacterId: characterId,
		NpcId:       npcId,
		Speaker:     speaker,
		Message:     message,
		Type:        npc2.CommandTypeSimple,
		Body:        npc2.CommandSimpleBody{Type: messageType},
	}
	return producer.SingleMessageProvider(key, value)
}
