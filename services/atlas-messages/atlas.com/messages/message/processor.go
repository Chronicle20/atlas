package message

import (
	"atlas-messages/character"
	"atlas-messages/command"
	message2 "atlas-messages/kafka/message/message"
	"atlas-messages/kafka/producer"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  *character.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) HandleGeneral(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, balloonOnly bool) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, worldId, channelId, c, message)
	if found {
		err = e(p.l)(p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
		}
		return err
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(generalChatEventProvider(worldId, channelId, mapId, actorId, message, balloonOnly))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *Processor) HandleMulti(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, chatType string, recipients []uint32) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, worldId, channelId, c, message)
	if found {
		err = e(p.l)(p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
		}
		return err
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(multiChatEventProvider(worldId, channelId, mapId, actorId, message, chatType, recipients))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *Processor) HandleWhisper(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, recipientName string) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	e, found := command.Registry().Get(p.l, p.ctx, worldId, channelId, c, message)
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

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(whisperChatEventProvider(worldId, channelId, mapId, actorId, message, tc.Id()))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *Processor) HandleMessenger(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, recipients []uint32) error {
	c, err := p.cp.GetById()(actorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character chatting [%d].", actorId)
		return err
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(messengerChatEventProvider(worldId, channelId, mapId, actorId, message, recipients))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
	}
	return err
}

func (p *Processor) HandlePet(worldId byte, channelId byte, mapId uint32, actorId uint32, message string, ownerId uint32, petSlot int8, nType byte, nAction byte, balloon bool) error {
	p.l.Debugf("Character [%d] pet [%d] sent message [%s].", ownerId, actorId, message)
	err := producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(petChatEventProvider(worldId, channelId, mapId, actorId, message, ownerId, petSlot, nType, nAction, balloon))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to relay message from character [%d] pet [%d].", ownerId, actorId)
	}
	return err
}
