package message

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func MultiChatTypeStrToInd(chatType string) byte {
	if chatType == ChatTypeBuddy {
		return 0
	}
	if chatType == ChatTypeParty {
		return 1
	}
	if chatType == ChatTypeGuild {
		return 2
	}
	if chatType == ChatTypeAlliance {
		return 3
	}
	return 99
}

func GeneralChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(generalChatCommandProvider(worldId, channelId, mapId, characterId, message, balloonOnly))
		}
	}
}

func BuddyChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(worldId, channelId, mapId, characterId, message, ChatTypeBuddy, recipients)
		}
	}
}

func PartyChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(worldId, channelId, mapId, characterId, message, ChatTypeParty, recipients)
		}
	}
}

func GuildChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(worldId, channelId, mapId, characterId, message, ChatTypeGuild, recipients)
		}
	}
}

func AllianceChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(worldId, channelId, mapId, characterId, message, ChatTypeAlliance, recipients)
		}
	}
}

func MultiChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(multiChatCommandProvider(worldId, channelId, mapId, characterId, message, chatType, recipients))
		}
	}
}
