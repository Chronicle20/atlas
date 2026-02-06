package npc

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enableActionsProvider(ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.StatusEvent[npc2.StatusEventStatChangedBody]{
		CharacterId: characterId,
		Type:        npc2.EventCharacterStatusTypeStatChanged,
		WorldId:     ch.WorldId(),
		Body: npc2.StatusEventStatChangedBody{
			ChannelId:       ch.Id(),
			ExclRequestSent: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func simpleConversationProvider(ch channel.Model, characterId uint32, npcId uint32, message string, messageType string, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandSimpleBody]{
		WorldId:        ch.WorldId(),
		ChannelId:      ch.Id(),
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

func numberConversationProvider(ch channel.Model, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandNumberBody]{
		WorldId:        ch.WorldId(),
		ChannelId:      ch.Id(),
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

func styleConversationProvider(ch channel.Model, characterId uint32, npcId uint32, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandStyleBody]{
		WorldId:        ch.WorldId(),
		ChannelId:      ch.Id(),
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

func slideMenuConversationProvider(ch channel.Model, characterId uint32, npcId uint32, message string, menuType uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationCommand[npc2.CommandSlideMenuBody]{
		WorldId:        ch.WorldId(),
		ChannelId:      ch.Id(),
		CharacterId:    characterId,
		NpcId:          npcId,
		Speaker:        speaker,
		EndChat:        endChat,
		SecondaryNpcId: secondaryNpcId,
		Message:        message,
		Type:           npc2.CommandTypeSlideMenu,
		Body: npc2.CommandSlideMenuBody{
			MenuType: menuType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
