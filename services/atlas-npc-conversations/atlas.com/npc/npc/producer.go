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
		WorldId:     worldId,
		Body: npc2.StatusEventStatChangedBody{
			ChannelId:       channelId,
			ExclRequestSent: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func simpleConversationProvider(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, messageType string, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandSimpleBody]{
		WorldId:        worldId,
		ChannelId:      channelId,
		CharacterId:    characterId,
		NpcId:          npcId,
		Speaker:        speaker,
		EndChat:        endChat,
		SecondaryNpcId: secondaryNpcId,
		Message:        message,
		Type:           npc2.CommandTypeSimple,
		Body:           npc2.CommandSimpleBody{Type: messageType},
	}
	return producer.SingleMessageProvider(key, value)
}

func numberConversationProvider(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandNumberBody]{
		WorldId:        worldId,
		ChannelId:      channelId,
		CharacterId:    characterId,
		NpcId:          npcId,
		Speaker:        speaker,
		EndChat:        endChat,
		SecondaryNpcId: secondaryNpcId,
		Message:        message,
		Type:           npc2.CommandTypeNumber,
		Body: npc2.CommandNumberBody{
			DefaultValue: def,
			MinValue:     min,
			MaxValue:     max,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func styleConversationProvider(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandStyleBody]{
		WorldId:        worldId,
		ChannelId:      channelId,
		CharacterId:    characterId,
		NpcId:          npcId,
		Speaker:        speaker,
		EndChat:        endChat,
		SecondaryNpcId: secondaryNpcId,
		Message:        message,
		Type:           npc2.CommandTypeStyle,
		Body: npc2.CommandStyleBody{
			Styles: styles,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
