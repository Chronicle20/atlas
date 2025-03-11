package message

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func generalChatCommandProvider(m _map.Model, actorId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatCommand[generalChatBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeGeneral,
		Body:      generalChatBody{BalloonOnly: balloonOnly},
	}
	return producer.SingleMessageProvider(key, value)
}

func multiChatCommandProvider(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatCommand[multiChatBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		ActorId:   actorId,
		Message:   message,
		Type:      chatType,
		Body:      multiChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func whisperChatCommandProvider(m _map.Model, actorId uint32, message string, recipientName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatCommand[whisperChatBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeWhisper,
		Body:      whisperChatBody{RecipientName: recipientName},
	}
	return producer.SingleMessageProvider(key, value)
}

func messengerChatCommandProvider(m _map.Model, actorId uint32, message string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := chatCommand[messengerChatBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		ActorId:   actorId,
		Message:   message,
		Type:      ChatTypeMessenger,
		Body:      messengerChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func petChatCommandProvider(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := chatCommand[petChatBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		ActorId:   uint32(petId),
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
