package system_message

import (
	"atlas-saga-orchestrator/kafka/message/system_message"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SendMessageCommandProvider creates a Kafka message for sending system messages to a character
func SendMessageCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, messageType string, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.SendMessageBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandSendMessage,
		Body: system_message.SendMessageBody{
			MessageType: messageType,
			Message:     message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// PlayPortalSoundCommandProvider creates a Kafka message for playing portal sound effect
func PlayPortalSoundCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.PlayPortalSoundBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandPlayPortalSound,
		Body:          system_message.PlayPortalSoundBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowInfoCommandProvider creates a Kafka message for showing info/tutorial effects
func ShowInfoCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, path string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.ShowInfoBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandShowInfo,
		Body: system_message.ShowInfoBody{
			Path: path,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowInfoTextCommandProvider creates a Kafka message for showing text messages
func ShowInfoTextCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, text string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.ShowInfoTextBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandShowInfoText,
		Body: system_message.ShowInfoTextBody{
			Text: text,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// UpdateAreaInfoCommandProvider creates a Kafka message for updating area info
func UpdateAreaInfoCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, area uint16, info string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.UpdateAreaInfoBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandUpdateAreaInfo,
		Body: system_message.UpdateAreaInfoBody{
			Area: area,
			Info: info,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowHintCommandProvider creates a Kafka message for showing a hint box
func ShowHintCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, hint string, width uint16, height uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.ShowHintBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandShowHint,
		Body: system_message.ShowHintBody{
			Hint:   hint,
			Width:  width,
			Height: height,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowGuideHintCommandProvider creates a Kafka message for showing a pre-defined guide hint by ID
func ShowGuideHintCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, hintId uint32, duration uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.ShowGuideHintBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandShowGuideHint,
		Body: system_message.ShowGuideHintBody{
			HintId:   hintId,
			Duration: duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowIntroCommandProvider creates a Kafka message for showing an intro/direction effect
func ShowIntroCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, path string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.ShowIntroBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandShowIntro,
		Body: system_message.ShowIntroBody{
			Path: path,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
