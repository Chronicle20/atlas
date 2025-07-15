package character

import (
	"atlas-parties/kafka/message"
	"atlas-parties/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	LoginAndEmit(worldId byte, channelId byte, mapId uint32, characterId uint32) error
	Login(mb *message.Buffer) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error
	LogoutAndEmit(characterId uint32) error
	Logout(mb *message.Buffer) func(characterId uint32) error
	ChannelChange(characterId uint32, channelId byte) error
	LevelChangeAndEmit(characterId uint32, level byte) error
	LevelChange(mb *message.Buffer) func(characterId uint32, level byte) error
	JobChangeAndEmit(characterId uint32, jobId job.Id) error
	JobChange(mb *message.Buffer) func(characterId uint32, jobId job.Id) error
	MapChange(characterId uint32, mapId uint32) error
	JoinParty(characterId uint32, partyId uint32) error
	LeaveParty(characterId uint32) error
	Delete(characterId uint32) error
	ByIdProvider(characterId uint32) model.Provider[Model]
	GetById(characterId uint32) (Model, error)
	GetForeignCharacterInfo(characterId uint32) (ForeignModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) LoginAndEmit(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Login(buf)(worldId, channelId, mapId, characterId)
	})

}

func (p *ProcessorImpl) Login(mb *message.Buffer) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
		c, err := p.GetById(characterId)
		if err != nil {
			p.l.Debugf("Adding character [%d] from world [%d] to registry.", characterId, worldId)
			fm, err := p.GetForeignCharacterInfo(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve needed character information from foreign service.")
				return err
			}
			c = GetRegistry().Create(p.t, worldId, channelId, mapId, characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
		}

		p.l.Debugf("Setting character [%d] to online in registry.", characterId)
		fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
		c = GetRegistry().Update(p.t, c.Id(), Model.Login, fn)

		if c.PartyId() != 0 {
			err = mb.Put(EnvEventMemberStatusTopic, loginEventProvider(c.PartyId(), c.WorldId(), characterId))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] logged in.", c.PartyId(), c.Id())
				return err
			}
		}

		return nil
	}
}

func (p *ProcessorImpl) LogoutAndEmit(characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Logout(buf)(characterId)
	})
}

func (p *ProcessorImpl) Logout(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		c, err := p.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
			return err
		}

		p.l.Debugf("Setting character [%d] to offline in registry.", characterId)
		c = GetRegistry().Update(p.t, c.Id(), Model.Logout)

		if c.PartyId() != 0 {
			err = mb.Put(EnvEventMemberStatusTopic, logoutEventProvider(c.PartyId(), c.WorldId(), characterId))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] logged out.", c.PartyId(), c.Id())
				return err
			}
		}

		return nil
	}
}

func (p *ProcessorImpl) ChannelChange(characterId uint32, channelId byte) error {
	c, err := p.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
		return err
	}

	p.l.Debugf("Setting character [%d] to be in channel [%d] in registry.", characterId, channelId)
	fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
	c = GetRegistry().Update(p.t, c.Id(), fn)
	return nil
}

func (p *ProcessorImpl) LevelChangeAndEmit(characterId uint32, level byte) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.LevelChange(buf)(characterId, level)
	})
}

func (p *ProcessorImpl) LevelChange(mb *message.Buffer) func(characterId uint32, level byte) error {
	return func(characterId uint32, level byte) error {
		c, err := p.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Warnf("Unable to locate character [%d] in registry for level change.", characterId)
			return err
		}
		
		oldLevel := c.Level()
		p.l.Debugf("Updating character [%d] level from [%d] to [%d] in registry.", characterId, oldLevel, level)
		fn := func(m Model) Model { return Model.ChangeLevel(m, level) }
		c = GetRegistry().Update(p.t, c.Id(), fn)
		
		// If character is in a party, emit party member level changed event
		if c.PartyId() != 0 {
			err = mb.Put(EnvEventMemberStatusTopic, levelChangedEventProvider(c.PartyId(), c.WorldId(), characterId, oldLevel, level, c.Name()))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] level changed.", c.PartyId(), c.Id())
				return err
			}
		}
		
		return nil
	}
}

func (p *ProcessorImpl) JobChangeAndEmit(characterId uint32, jobId job.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.JobChange(buf)(characterId, jobId)
	})
}

func (p *ProcessorImpl) JobChange(mb *message.Buffer) func(characterId uint32, jobId job.Id) error {
	return func(characterId uint32, jobId job.Id) error {
		c, err := p.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Warnf("Unable to locate character [%d] in registry for job change.", characterId)
			return err
		}
		
		oldJobId := c.JobId()
		p.l.Debugf("Updating character [%d] job from [%d] to [%d] in registry.", characterId, oldJobId, jobId)
		fn := func(m Model) Model { return Model.ChangeJob(m, jobId) }
		c = GetRegistry().Update(p.t, c.Id(), fn)
		
		// If character is in a party, emit party member job changed event
		if c.PartyId() != 0 {
			err = mb.Put(EnvEventMemberStatusTopic, jobChangedEventProvider(c.PartyId(), c.WorldId(), characterId, oldJobId, jobId, c.Name()))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] job changed.", c.PartyId(), c.Id())
				return err
			}
		}
		
		return nil
	}
}

func (p *ProcessorImpl) MapChange(characterId uint32, mapId uint32) error {
	c, err := p.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
		return err
	}

	p.l.Debugf("Setting character [%d] to be in map [%d] in registry.", characterId, mapId)
	fn := func(m Model) Model { return Model.ChangeMap(m, mapId) }
	c = GetRegistry().Update(p.t, c.Id(), fn)
	return nil
}

func (p *ProcessorImpl) JoinParty(characterId uint32, partyId uint32) error {
	c, err := p.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
		return err
	}

	p.l.Debugf("Setting character [%d] to be in party [%d] in registry.", characterId, partyId)
	fn := func(m Model) Model { return Model.JoinParty(m, partyId) }
	c = GetRegistry().Update(p.t, c.Id(), fn)
	return nil
}

func (p *ProcessorImpl) LeaveParty(characterId uint32) error {
	c, err := GetRegistry().Get(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
		return err
	}

	p.l.Debugf("Setting character [%d] to no longer have a party in the registry.", characterId)
	c = GetRegistry().Update(p.t, c.Id(), Model.LeaveParty)
	return nil
}

func (p *ProcessorImpl) Delete(characterId uint32) error {
	c, err := GetRegistry().Get(p.t, characterId)
	if err != nil {
		p.l.Warnf("Character [%d] not found in registry, may have already been deleted.", characterId)
		return nil
	}

	if c.PartyId() != 0 {
		p.l.Debugf("Character [%d] was in party [%d], leaving party before deletion.", characterId, c.PartyId())
		err = p.LeaveParty(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to remove character [%d] from party [%d] before deletion.", characterId, c.PartyId())
			return err
		}
	}

	p.l.Debugf("Removing character [%d] from registry.", characterId)
	err = GetRegistry().Delete(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to delete character [%d] from registry.", characterId)
		return err
	}

	return nil
}

func (p *ProcessorImpl) ByIdProvider(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		c, err := GetRegistry().Get(p.t, characterId)
		if errors.Is(err, ErrNotFound) {
			fm, ferr := p.GetForeignCharacterInfo(characterId)
			if ferr != nil {
				return Model{}, err
			}
			c = GetRegistry().Create(p.t, fm.WorldId(), 0, fm.MapId(), characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
		}
		return c, nil
	}
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return p.ByIdProvider(characterId)()
}

func (p *ProcessorImpl) GetForeignCharacterInfo(characterId uint32) (ForeignModel, error) {
	return requests.Provider[ForeignRestModel, ForeignModel](p.l, p.ctx)(requestById(characterId), ExtractForeign)()
}
