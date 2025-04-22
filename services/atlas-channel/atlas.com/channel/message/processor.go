package message

import (
	message2 "atlas-channel/kafka/message/message"
	"atlas-channel/kafka/producer"
	message3 "atlas-channel/kafka/producer/message"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
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

func (p *Processor) GeneralChat(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(message3.GeneralChatCommandProvider(m, actorId, message, balloonOnly))
}

func (p *Processor) BuddyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeBuddy, recipients)
}

func (p *Processor) PartyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeParty, recipients)
}

func (p *Processor) GuildChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeGuild, recipients)
}

func (p *Processor) AllianceChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, message2.ChatTypeAlliance, recipients)
}

func (p *Processor) MultiChat(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(message3.MultiChatCommandProvider(m, actorId, message, chatType, recipients))
}

func (p *Processor) WhisperChat(m _map.Model, actorId uint32, message string, recipientName string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(message3.WhisperChatCommandProvider(m, actorId, message, recipientName))
}

func (p *Processor) MessengerChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(message3.MessengerChatCommandProvider(m, actorId, message, recipients))
}

func (p *Processor) PetChat(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(message2.EnvCommandTopicChat)(message3.PetChatCommandProvider(m, petId, message, ownerId, petSlot, nType, nAction, balloon))
}
