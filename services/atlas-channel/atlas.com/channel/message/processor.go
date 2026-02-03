package message

import (
	message2 "atlas-channel/kafka/message/message"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for message processing
type Processor interface {
	GeneralChat(f field.Model, actorId uint32, message string, balloonOnly bool) error
	BuddyChat(f field.Model, actorId uint32, message string, recipients []uint32) error
	PartyChat(f field.Model, actorId uint32, message string, recipients []uint32) error
	GuildChat(f field.Model, actorId uint32, message string, recipients []uint32) error
	AllianceChat(f field.Model, actorId uint32, message string, recipients []uint32) error
	MultiChat(f field.Model, actorId uint32, message string, chatType string, recipients []uint32) error
	WhisperChat(f field.Model, actorId uint32, message string, recipientName string) error
	MessengerChat(f field.Model, actorId uint32, message string, recipients []uint32) error
	PetChat(f field.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error
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

func (p *ProcessorImpl) GeneralChat(f field.Model, actorId uint32, message string, balloonOnly bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(GeneralChatCommandProvider(f, actorId, message, balloonOnly))
}

func (p *ProcessorImpl) BuddyChat(f field.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(f, actorId, message, message2.ChatTypeBuddy, recipients)
}

func (p *ProcessorImpl) PartyChat(f field.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(f, actorId, message, message2.ChatTypeParty, recipients)
}

func (p *ProcessorImpl) GuildChat(f field.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(f, actorId, message, message2.ChatTypeGuild, recipients)
}

func (p *ProcessorImpl) AllianceChat(f field.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(f, actorId, message, message2.ChatTypeAlliance, recipients)
}

func (p *ProcessorImpl) MultiChat(f field.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(MultiChatCommandProvider(f, actorId, message, chatType, recipients))
}

func (p *ProcessorImpl) WhisperChat(f field.Model, actorId uint32, message string, recipientName string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(WhisperChatCommandProvider(f, actorId, message, recipientName))
}

func (p *ProcessorImpl) MessengerChat(f field.Model, actorId uint32, message string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(MessengerChatCommandProvider(f, actorId, message, recipients))
}

func (p *ProcessorImpl) PetChat(f field.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(PetChatCommandProvider(f, petId, message, ownerId, petSlot, nType, nAction, balloon))
}
