package message

import (
	"atlas-messages/character"
	"atlas-messages/chat"
	"atlas-messages/command"
	message2 "atlas-messages/kafka/message/message"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	HandleGeneral(f field.Model, actorId uint32, message string, balloonOnly bool) error
	HandleMulti(f field.Model, actorId uint32, message string, chatType string, recipients []uint32) error
	HandleWhisper(f field.Model, actorId uint32, message string, recipientName string) error
	HandleMessenger(f field.Model, actorId uint32, message string, recipients []uint32) error
	HandlePet(f field.Model, actorId uint32, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error
	IssuePinkText(f field.Model, actorId uint32, message string, recipients []uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return NewProcessorWithClients(l, ctx, character.NewProcessor(l, ctx))
}

// NewProcessorWithClients constructs a Processor with an explicit
// character.Processor implementation. Production callers use NewProcessor;
// callers that already hold a character.Processor (or a substitute, e.g.
// tests) inject it here.
func NewProcessorWithClients(l logrus.FieldLogger, ctx context.Context, cp character.Processor) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  cp,
	}
}

func (p *ProcessorImpl) HandleGeneral(f field.Model, actorId uint32, message string, balloonOnly bool) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, f, c, message)
	if found {
		err = e(p.l)(p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
		}
		return err
	}

	p.captureLine(f, actorId, c.Name(), message2.ChatTypeGeneral, message)

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(generalChatEventProvider(f, actorId, message, balloonOnly))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *ProcessorImpl) HandleMulti(f field.Model, actorId uint32, message string, chatType string, recipients []uint32) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, f, c, message)
	if found {
		err = e(p.l)(p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
		}
		return err
	}

	p.captureLine(f, actorId, c.Name(), chatType, message)

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(multiChatEventProvider(f, actorId, message, chatType, recipients))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *ProcessorImpl) HandleWhisper(f field.Model, actorId uint32, message string, recipientName string) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, f, c, message)
	if found {
		err = e(p.l)(p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
		}
		return err
	}

	tc, err := p.cp.GetByName()(recipientName)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate recipient [%s].", recipientName)
		return err
	}

	if c.WorldId() != tc.WorldId() {
		return errors.New("not in world")
	}

	p.captureLine(f, actorId, c.Name(), message2.ChatTypeWhisper, message)

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(whisperChatEventProvider(f, actorId, message, tc.Id()))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *ProcessorImpl) HandleMessenger(f field.Model, actorId uint32, message string, recipients []uint32) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	p.captureLine(f, actorId, c.Name(), message2.ChatTypeMessenger, message)

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(messengerChatEventProvider(f, actorId, message, recipients))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *ProcessorImpl) HandlePet(f field.Model, actorId uint32, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	p.l.Debugf("Character [%d] pet [%d] sent message [%s].", ownerId, actorId, message)
	err := producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(petChatEventProvider(f, actorId, message, ownerId, petSlot, nType, nAction, balloon))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d] pet [%d].", ownerId, actorId)
	}
	return err
}

func (p *ProcessorImpl) IssuePinkText(f field.Model, actorId uint32, message string, recipients []uint32) error {
	err := producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(pinkTextChatEventProvider(f, actorId, message, recipients))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from actorId [%d].", actorId)
	}
	return err
}

// captureLine records a player-authored chat line for report corroboration.
// Best-effort: a Redis outage logs a warning and never blocks the chat flow.
func (p *ProcessorImpl) captureLine(f field.Model, senderId uint32, senderName string, chatType string, text string) {
	if err := chat.NewProcessor(p.l, p.ctx).Capture(f, senderId, senderName, chatType, text); err != nil {
		p.l.WithError(err).Warnf("Unable to capture chat line for character [%d].", senderId)
	}
}
