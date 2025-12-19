package party

import (
	"atlas-parties/character"
	"atlas-parties/invite"
	"atlas-parties/kafka/message"
	"atlas-parties/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const StartPartyId = uint32(1000000000)

var ErrNotFound = errors.New("not found")
var ErrAtCapacity = errors.New("at capacity")
var ErrAlreadyIn = errors.New("already in party")
var ErrNotIn = errors.New("not in party")
var ErrNotAsBeginner = errors.New("not as beginner")
var ErrNotAsGm = errors.New("not as gm")

type Processor interface {
	AllProvider() ([]Model, error)
	ByIdProvider(partyId uint32) model.Provider[Model]
	GetSlice(filters ...model.Filter[Model]) ([]Model, error)
	GetById(partyId uint32) (Model, error)
	GetByCharacter(characterId uint32) (Model, error)
	ByCharacterProvider(characterId uint32) model.Provider[Model]

	Create(mb *message.Buffer) func(leaderId uint32) (Model, error)
	CreateAndEmit(leaderId uint32) (Model, error)
	Join(mb *message.Buffer) func(partyId uint32, characterId uint32) (Model, error)
	JoinAndEmit(partyId uint32, characterId uint32) (Model, error)
	Expel(mb *message.Buffer) func(actorId uint32, partyId uint32, characterId uint32) (Model, error)
	ExpelAndEmit(actorId uint32, partyId uint32, characterId uint32) (Model, error)
	Leave(mb *message.Buffer) func(partyId uint32, characterId uint32) (Model, error)
	LeaveAndEmit(partyId uint32, characterId uint32) (Model, error)
	ChangeLeader(mb *message.Buffer) func(actorId uint32, partyId uint32, characterId uint32) (Model, error)
	ChangeLeaderAndEmit(actorId uint32, partyId uint32, characterId uint32) (Model, error)
	RequestInvite(mb *message.Buffer) func(actorId uint32, characterId uint32) error
	RequestInviteAndEmit(actorId uint32, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	cp  character.Processor
	ip  invite.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		cp:  character.NewProcessor(l, ctx),
		ip:  invite.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) AllProvider() ([]Model, error) {
	return GetRegistry().GetAll(p.t), nil
}

func (p *ProcessorImpl) ByIdProvider(partyId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetRegistry().Get(p.t, partyId)
	}
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, id := range m.members {
			if id == memberId {
				return true
			}
		}
		return false
	}
}

func (p *ProcessorImpl) GetSlice(filters ...model.Filter[Model]) ([]Model, error) {
	return model.FilteredProvider(p.AllProvider, model.Filters[Model](filters...))()
}

func (p *ProcessorImpl) GetById(partyId uint32) (Model, error) {
	return p.ByIdProvider(partyId)()
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) (Model, error) {
	return p.ByCharacterProvider(characterId)()
}

// Efficient provider for character-to-party lookup
func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetRegistry().GetPartyByCharacter(p.t, characterId)
	}
}

func (p *ProcessorImpl) CreateAndEmit(leaderId uint32) (Model, error) {
	var party Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		party, err = p.Create(buf)(leaderId)
		return err
	})
	return party, err
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(leaderId uint32) (Model, error) {
	return func(leaderId uint32) (Model, error) {
		c, err := p.cp.GetById(leaderId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", leaderId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
			}
			return Model{}, err
		}

		if c.PartyId() != 0 {
			p.l.Errorf("Character [%d] already in party. Cannot create another one.", leaderId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined1, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
			}
			return Model{}, ErrAlreadyIn
		}

		if job.IsBeginner(c.JobId()) {
			p.l.Errorf("Character [%d] is a beginner, cannot create parties.", leaderId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeBeginnerCannotCreate, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
			}
			return Model{}, ErrNotAsBeginner
		}

		if c.GM() > 0 {
			p.l.Errorf("Character [%d] is a GM, cannot create parties.", leaderId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeGmCannotCreate, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
			}
			return Model{}, ErrNotAsGm
		}

		party := GetRegistry().Create(p.t, leaderId)

		p.l.Debugf("Created party [%d] for leader [%d].", party.Id(), leaderId)

		err = mb.Put(EnvEventStatusTopic, createdEventProvider(leaderId, party.Id(), c.WorldId()))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to announce the party [%d] was created.", leaderId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", party.Id())
			}
			return Model{}, err
		}

		err = p.cp.JoinParty(leaderId, party.Id())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to have character [%d] join party [%d]", leaderId, party.Id())
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", party.Id())
			}
			return Model{}, err
		}

		return party, nil
	}
}

func (p *ProcessorImpl) JoinAndEmit(partyId uint32, characterId uint32) (Model, error) {
	var party Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		party, err = p.Join(buf)(partyId, characterId)
		return err
	})
	return party, err
}

func (p *ProcessorImpl) Join(mb *message.Buffer) func(partyId uint32, characterId uint32) (Model, error) {
	return func(partyId uint32, characterId uint32) (Model, error) {
		c, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		if c.PartyId() != 0 {
			p.l.Errorf("Character [%d] already in party. Cannot create another one.", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, c.PartyId(), c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined2, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrAlreadyIn
		}

		party, err := GetRegistry().Get(p.t, partyId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		if len(party.Members()) >= 6 {
			p.l.Errorf("Party [%d] already at capacity.", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorTypeAtCapacity, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrAtCapacity
		}

		fn := func(m Model) Model { return Model.AddMember(m, characterId) }
		party, err = GetRegistry().Update(p.t, partyId, fn)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to join party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}
		err = p.cp.JoinParty(characterId, partyId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to join party [%d].", partyId)
			party, err = GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				p.l.WithError(err).Errorf("Unable to clean up party [%d], when failing to add member [%d].", partyId, characterId)
			}
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		p.l.Debugf("Character [%d] joined party [%d].", characterId, partyId)
		err = mb.Put(EnvEventStatusTopic, joinedEventProvider(characterId, party.Id(), c.WorldId()))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to announce the party [%d] was created.", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		return party, nil
	}
}

func (p *ProcessorImpl) ExpelAndEmit(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	var party Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		party, err = p.Expel(buf)(actorId, partyId, characterId)
		return err
	})
	return party, err
}

func (p *ProcessorImpl) Expel(mb *message.Buffer) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		c, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		if c.PartyId() != partyId {
			p.l.Errorf("Character [%d] not in party.", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		party, err := GetRegistry().Get(p.t, partyId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		party, err = GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
		if err != nil {
			p.l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}
		err = p.cp.LeaveParty(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
			party, err = GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
			if err != nil {
				p.l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
			}
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		p.l.Debugf("Character [%d] expelled from party [%d].", characterId, partyId)

		// Check if party is empty after expulsion and disband if necessary
		if len(party.Members()) == 0 {
			p.l.Debugf("Party [%d] is empty after expelling character [%d], disbanding party.", partyId, characterId)

			// Emit disband event before removing party
			err = mb.Put(EnvEventStatusTopic, disbandEventProvider(actorId, party.Id(), c.WorldId(), []uint32{}))
			if err != nil {
				p.l.WithError(err).Warnf("Unable to emit disband event for party [%d].", partyId)
				// Don't return error as the disbanding will still proceed
			} else {
				p.l.Infof("Emitted disband event for party [%d] due to last member [%d] expulsion.", partyId, characterId)
			}

			// Party is empty, disband it
			GetRegistry().Remove(p.t, partyId)
			p.l.Infof("Party [%d] disbanded after expelling last member [%d].", partyId, characterId)

			return Model{}, nil
		}
		// Party still has members, emit expel event normally
		err = mb.Put(EnvEventStatusTopic, expelEventProvider(actorId, party.Id(), c.WorldId(), characterId))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to announce the party [%d] was left.", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		return party, nil
	}
}

func (p *ProcessorImpl) LeaveAndEmit(partyId uint32, characterId uint32) (Model, error) {
	var party Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		party, err = p.Leave(buf)(partyId, characterId)
		return err
	})
	return party, err
}

func (p *ProcessorImpl) Leave(mb *message.Buffer) func(partyId uint32, characterId uint32) (Model, error) {
	return func(partyId uint32, characterId uint32) (Model, error) {
		c, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		if c.PartyId() != partyId {
			p.l.Errorf("Character [%d] not in party.", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		party, err := GetRegistry().Get(p.t, partyId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		var disbandParty = party.LeaderId() == characterId

		party, err = GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
		if err != nil {
			p.l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}
		err = p.cp.LeaveParty(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
			party, err = GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
			if err != nil {
				p.l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
			}
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", characterId)
			}
			return Model{}, err
		}

		if disbandParty {
			for _, m := range party.Members() {
				err = p.cp.LeaveParty(m)
				if err != nil {
					p.l.WithError(err).Errorf("Character [%d] stuck in party [%d].", m, partyId)
				}
			}

			GetRegistry().Remove(p.t, partyId)
			p.l.Debugf("Party [%d] has been disbanded.", partyId)
			err = mb.Put(EnvEventStatusTopic, disbandEventProvider(characterId, partyId, c.WorldId(), party.Members()))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] was disbanded.", partyId)
				err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}
		} else {
			p.l.Debugf("Character [%d] left party [%d].", characterId, partyId)
			err = mb.Put(EnvEventStatusTopic, leftEventProvider(characterId, partyId, c.WorldId()))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] was left.", partyId)
				err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}
		}

		return party, nil
	}
}

func (p *ProcessorImpl) ChangeLeaderAndEmit(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	var party Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		party, err = p.ChangeLeader(buf)(actorId, partyId, characterId)
		return err
	})
	return party, err
}

func (p *ProcessorImpl) ChangeLeader(mb *message.Buffer) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		a, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", actorId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, a.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		c, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, err
		}

		if c.PartyId() != partyId {
			p.l.Errorf("Character [%d] not in party. Cannot become leader.", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		if a.PartyId() != c.PartyId() {
			p.l.Errorf("Character [%d] not in the same party as actor [%d]. Cannot become leader.", characterId, actorId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		if a.WorldId() != c.WorldId() && a.ChannelId() != c.ChannelId() {
			p.l.Errorf("Character [%d] not in the same channel as actor [%d]. Cannot become leader.", characterId, actorId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorTypeNotInChannel, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		if a.MapId() != c.MapId() {
			p.l.Errorf("Character [%d] not in the same map as actor [%d]. Cannot become leader.", characterId, actorId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorTypeUnableToInVicinity, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
			return Model{}, ErrNotIn
		}

		party, err := GetRegistry().Update(p.t, partyId, func(m Model) Model { return Model.SetLeader(m, characterId) })
		if err != nil {
			p.l.WithError(err).Errorf("Unable to join party [%d].", partyId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
		}

		p.l.Debugf("Character [%d] became leader of party [%d].", characterId, partyId)
		err = mb.Put(EnvEventStatusTopic, changeLeaderEventProvider(actorId, party.Id(), c.WorldId(), characterId))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to announce leadership change in party [%d].", c.Id())
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
			}
		}
		return party, nil
	}
}

func (p *ProcessorImpl) RequestInviteAndEmit(actorId uint32, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.RequestInvite(buf)(actorId, characterId)
	})
}

func (p *ProcessorImpl) RequestInvite(mb *message.Buffer) func(actorId uint32, characterId uint32) error {
	return func(actorId uint32, characterId uint32) error {
		a, err := p.cp.GetById(actorId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", actorId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, 0, a.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", 0)
			}
			return err
		}

		c, err := p.cp.GetById(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Error getting character [%d].", characterId)
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, a.PartyId(), c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
			}
			return err
		}

		if c.PartyId() != 0 {
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, c.PartyId(), c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined2, c.Name()))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
			}
			return ErrAlreadyIn
		}

		var party Model
		if a.PartyId() == 0 {
			party, err = p.Create(mb)(actorId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to automatically create party [%d].", a.PartyId())
				return err
			}
		} else {
			party, err = p.GetById(a.PartyId())
			if err != nil {
				return err
			}
		}

		if len(party.Members()) >= 6 {
			p.l.Errorf("Party [%d] already at capacity.", party.Id())
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, party.Id(), c.WorldId(), EventPartyStatusErrorTypeAtCapacity, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", party.Id())
			}
			return ErrAtCapacity
		}

		err = p.ip.Create(actorId, a.WorldId(), party.Id(), characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to announce party [%d] invite.", party.Id())
			err = mb.Put(EnvEventStatusTopic, errorEventProvider(actorId, a.PartyId(), c.WorldId(), EventPartyStatusErrorUnexpected, ""))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
			}
			return err
		}

		return nil
	}
}
