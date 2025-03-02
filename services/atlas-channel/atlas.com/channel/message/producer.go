package message

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func generalChatCommandProvider(m _map.Model, characterId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := chatCommand[generalChatBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		CharacterId: characterId,
		Message:     message,
		Type:        ChatTypeGeneral,
		Body:        generalChatBody{BalloonOnly: balloonOnly},
	}
	return producer.SingleMessageProvider(key, value)
}

func multiChatCommandProvider(m _map.Model, characterId uint32, message string, chatType string, recipients []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := chatCommand[multiChatBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		CharacterId: characterId,
		Message:     message,
		Type:        chatType,
		Body:        multiChatBody{Recipients: recipients},
	}
	return producer.SingleMessageProvider(key, value)
}

func whisperChatCommandProvider(m _map.Model, characterId uint32, message string, recipientName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := chatCommand[whisperChatBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		CharacterId: characterId,
		Message:     message,
		Type:        ChatTypeWhisper,
		Body:        whisperChatBody{RecipientName: recipientName},
	}
	return producer.SingleMessageProvider(key, value)
}
