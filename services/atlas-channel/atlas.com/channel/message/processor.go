package message

import (
	message2 "atlas-channel/kafka/message/message"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for message processing
type Processor interface {
	GeneralChat(m _map.Model, actorId uint32, message string, balloonOnly bool) error
	BuddyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error
	PartyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error
	GuildChat(m _map.Model, actorId uint32, message string, recipients []uint32) error
	AllianceChat(m _map.Model, actorId uint32, message string, recipients []uint32) error
	MultiChat(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error
	WhisperChat(m _map.Model, actorId uint32, message string, recipientName string) error
	MessengerChat(m _map.Model, actorId uint32, message string, recipients []uint32) error
	PetChat(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func MultiChatTypeStrToInd(chatType string) byte {
	if chatType == message2.ChatTypeBuddy {
		return 0
	}
	if chatType == message2.ChatTypeParty {
		return 1
	}
	if chatType == message2.ChatTypeGuild {
		return 2
	}
	if chatType == message2.ChatTypeAlliance {
		return 3
	}
	return 99
}

func (p *ProcessorImpl) GeneralChat(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(GeneralChatCommandProvider(m, actorId, message, balloonOnly))
}

func (p *ProcessorImpl) BuddyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeBuddy, recipients)
}

func (p *ProcessorImpl) PartyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeParty, recipients)
}

func (p *ProcessorImpl) GuildChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeGuild, recipients)
}

func (p *ProcessorImpl) AllianceChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeAlliance, recipients)
}

func (p *ProcessorImpl) MultiChat(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(MultiChatCommandProvider(m, actorId, message, chatType, recipients))
}

func (p *ProcessorImpl) WhisperChat(m _map.Model, actorId uint32, message string, recipientName string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(WhisperChatCommandProvider(m, actorId, message, recipientName))
}

func (p *ProcessorImpl) MessengerChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(MessengerChatCommandProvider(m, actorId, message, recipients))
}

func (p *ProcessorImpl) PetChat(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(PetChatCommandProvider(m, petId, message, ownerId, petSlot, nType, nAction, balloon))
}
