package message

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func generalChatEventProvider(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatEvent[generalChatBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeGeneral,
		Body:      generalChatBody{BalloonOnly: balloonOnly},
	}
	return producer.SingleMessageProvider(key, value)
}

func multiChatEventProvider(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, chatType string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatEvent[multiChatBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ActorId:   actorId,
		Message:   message,
		Type:      chatType,
		Body:      multiChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func whisperChatEventProvider(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, recipient uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatEvent[whisperChatBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeWhisper,
		Body:      whisperChatBody{Recipient: recipient},
	}
	return producer.SingleMessageProvider(key, value)
}

func messengerChatEventProvider(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatEvent[messengerChatBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeMessenger,
		Body:      messengerChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func petChatEventProvider(worldId byte, channelId byte, mapId uint32, petId uint32, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := chatEvent[petChatBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ActorId:   petId,
		Message:   message,
		Type:      ChatTypePet,
		Body: petChatBody{
			OwnerId: ownerId,
			PetSlot: petSlot,
			Type:    nType,
			Action:  nAction,
			Balloon: balloon,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
