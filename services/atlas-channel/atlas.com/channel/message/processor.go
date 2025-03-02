package message

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func GeneralChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, balloonOnly bool) error {
		return func(m _map.Model, characterId uint32, message string, balloonOnly bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(generalChatCommandProvider(m, characterId, message, balloonOnly))
		}
	}
}

func BuddyChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, characterId, message, ChatTypeBuddy, recipients)
		}
	}
}

func PartyChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, characterId, message, ChatTypeParty, recipients)
		}
	}
}

func GuildChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, characterId, message, ChatTypeGuild, recipients)
		}
	}
}

func AllianceChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, characterId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, characterId, message, ChatTypeAlliance, recipients)
		}
	}
}

func MultiChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, chatType string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, chatType string, recipients []uint32) error {
		return func(m _map.Model, characterId uint32, message string, chatType string, recipients []uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(multiChatCommandProvider(m, characterId, message, chatType, recipients))
		}
	}
}

func WhisperChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipientName string) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string, recipientName string) error {
		return func(m _map.Model, characterId uint32, message string, recipientName string) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(whisperChatCommandProvider(m, characterId, message, recipientName))
		}
	}
}
