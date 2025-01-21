package npc

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enableActionsProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventStatChangedBody]{
		CharacterId: characterId,
		Type:        EventCharacterStatusTypeStatChanged,
		WorldId:     worldId,
		Body: statusEventStatChangedBody{
			ChannelId:       channelId,
			ExclRequestSent: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func simpleConversationProvider(worldId byte, channelId byte, characterId uint32, npcId uint32, message string, messageType string, speaker string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[commandSimpleBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		CharacterId: characterId,
		NpcId:       npcId,
		Speaker:     speaker,
		Message:     message,
		Type:        CommandTypeSimple,
		Body:        commandSimpleBody{Type: messageType},
	}
	return producer.SingleMessageProvider(key, value)
}
