package message

import (
	"atlas-channel/kafka/producer"
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

func (p *Processor) GeneralChat(m _map.Model, actorId uint32, message string, balloonOnly bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicChat)(generalChatCommandProvider(m, actorId, message, balloonOnly))
}

func (p *Processor) BuddyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, ChatTypeBuddy, recipients)
}

func (p *Processor) PartyChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, ChatTypeParty, recipients)
}

func (p *Processor) GuildChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, ChatTypeGuild, recipients)
}

func (p *Processor) AllianceChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return p.MultiChat(m, actorId, message, ChatTypeAlliance, recipients)
}

func (p *Processor) MultiChat(m _map.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicChat)(multiChatCommandProvider(m, actorId, message, chatType, recipients))
}

func (p *Processor) WhisperChat(m _map.Model, actorId uint32, message string, recipientName string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicChat)(whisperChatCommandProvider(m, actorId, message, recipientName))
}

func (p *Processor) MessengerChat(m _map.Model, actorId uint32, message string, recipients []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicChat)(messengerChatCommandProvider(m, actorId, message, recipients))
}

func (p *Processor) PetChat(m _map.Model, petId uint64, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicChat)(petChatCommandProvider(m, petId, message, ownerId, petSlot, nType, nAction, balloon))
}
