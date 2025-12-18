package message

import (
	message2 "atlas-channel/kafka/message/message"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func GeneralChatCommandProvider(m _map.Model, actorId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.GeneralChatBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeGeneral,
		Body:      message2.GeneralChatBody{BalloonOnly: balloonOnly},
	}
	return producer.SingleMessageProvider(key, value)
}

func MultiChatCommandProvider(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.MultiChatBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		ActorId:   actorId,
		Message:   message,
		Type:      chatType,
		Body:      message2.MultiChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func WhisperChatCommandProvider(m _map.Model, actorId uint32, message string, recipientName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.WhisperChatEventBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeWhisper,
		Body:      message2.WhisperChatEventBody{RecipientName: recipientName},
	}
	return producer.SingleMessageProvider(key, value)
}

func MessengerChatCommandProvider(m _map.Model, actorId uint32, message string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.MessengerChatBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeMessenger,
		Body:      message2.MessengerChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func PetChatCommandProvider(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := message2.Command[message2.PetChatBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		ActorId:   uint32(petId),
		Message:   message,
		Type:      message2.ChatTypePet,
		Body: message2.PetChatBody{
			OwnerId: ownerId,
			PetSlot: petSlot,
			Type:    nType,
			Action:  nAction,
			Balloon: balloon,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
