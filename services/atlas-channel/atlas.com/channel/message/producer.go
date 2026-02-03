package message

import (
	message2 "atlas-channel/kafka/message/message"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func GeneralChatCommandProvider(f field.Model, actorId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.GeneralChatBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeGeneral,
		Body:      message2.GeneralChatBody{BalloonOnly: balloonOnly},
	}
	return producer.SingleMessageProvider(key, value)
}

func MultiChatCommandProvider(f field.Model, actorId uint32, message string, chatType string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.MultiChatBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		ActorId:   actorId,
		Message:   message,
		Type:      chatType,
		Body:      message2.MultiChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func WhisperChatCommandProvider(f field.Model, actorId uint32, message string, recipientName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.WhisperChatEventBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeWhisper,
		Body:      message2.WhisperChatEventBody{RecipientName: recipientName},
	}
	return producer.SingleMessageProvider(key, value)
}

func MessengerChatCommandProvider(f field.Model, actorId uint32, message string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := message2.Command[message2.MessengerChatBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		ActorId:   actorId,
		Message:   message,
		Type:      message2.ChatTypeMessenger,
		Body:      message2.MessengerChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func PetChatCommandProvider(f field.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := message2.Command[message2.PetChatBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
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
