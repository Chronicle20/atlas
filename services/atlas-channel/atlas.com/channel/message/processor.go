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

func GeneralChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
		return func(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(generalChatCommandProvider(m, actorId, message, balloonOnly))
		}
	}
}

func BuddyChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, actorId, message, ChatTypeBuddy, recipients)
		}
	}
}

func PartyChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, actorId, message, ChatTypeParty, recipients)
		}
	}
}

func GuildChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, actorId, message, ChatTypeGuild, recipients)
		}
	}
}

func AllianceChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
			return MultiChat(l)(ctx)(m, actorId, message, ChatTypeAlliance, recipients)
		}
	}
}

func MultiChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(multiChatCommandProvider(m, actorId, message, chatType, recipients))
		}
	}
}

func WhisperChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipientName string) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipientName string) error {
		return func(m _map.Model, actorId uint32, message string, recipientName string) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(whisperChatCommandProvider(m, actorId, message, recipientName))
		}
	}
}

func MessengerChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
		return func(m _map.Model, actorId uint32, message string, recipients []uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(messengerChatCommandProvider(m, actorId, message, recipients))
		}
	}
}

func PetChat(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	return func(ctx context.Context) func(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
		return func(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicChat)(petChatCommandProvider(m, petId, message, ownerId, petSlot, nType, nAction, balloon))
		}
	}
}
