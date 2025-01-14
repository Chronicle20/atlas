package party

import (
	"atlas-parties/character"
	"atlas-parties/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const StartPartyId = uint32(1000000000)

var ErrNotFound = errors.New("not found")
var ErrAtCapacity = errors.New("at capacity")
var ErrAlreadyIn = errors.New("already in party")
var ErrNotIn = errors.New("not in party")

func allProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().GetAll(t), nil
	}
}

func byIdProvider(ctx context.Context) func(partyId uint32) model.Provider[Model] {
	return func(partyId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetRegistry().Get(t, partyId)
		}
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

func GetSlice(ctx context.Context) func(filters ...model.Filter[Model]) ([]Model, error) {
	return func(filters ...model.Filter[Model]) ([]Model, error) {
		return model.FilteredProvider(allProvider(ctx), model.Filters[Model](filters...))()
	}
}

func GetById(ctx context.Context) func(partyId uint32) (Model, error) {
	return func(partyId uint32) (Model, error) {
		return byIdProvider(ctx)(partyId)()
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(leaderId uint32) (Model, error) {
	return func(ctx context.Context) func(leaderId uint32) (Model, error) {
		return func(leaderId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(leaderId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", leaderId)
				return Model{}, err
			}

			if c.PartyId() != 0 {
				l.Errorf("Character [%d] already in party. Cannot create another one.", leaderId)
				return Model{}, ErrAlreadyIn
			}

			p := GetRegistry().Create(t, leaderId)

			l.Debugf("Created party [%d] for leader [%d].", p.Id(), leaderId)

			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdEventProvider(leaderId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was created.", c.Id())
				return Model{}, err
			}

			err = character.JoinParty(l)(ctx)(leaderId, p.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to have character [%d] join party [%d]", leaderId, p.Id())
				return Model{}, err
			}

			//l.Debugf("Character [%d] joined party [%d].", leaderId, p.Id())
			//err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(joinedEventProvider(p.Id(), c.WorldId(), leaderId))
			//if err != nil {
			//	l.WithError(err).Errorf("Unable to announce the party [%d] was created.", c.Id())
			//	return Model{}, err
			//}

			return p, nil
		}
	}
}

func Join(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
		return func(partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				return Model{}, err
			}

			if c.PartyId() != 0 {
				l.Errorf("Character [%d] already in party. Cannot join another one.", characterId)
				return Model{}, ErrAlreadyIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				return Model{}, err
			}

			if len(p.members) > 6 {
				l.Errorf("Party [%d] already at capacity.", partyId)
				return Model{}, ErrAtCapacity
			}

			fn := func(m Model) Model { return Model.AddMember(m, characterId) }
			p, err = GetRegistry().Update(t, partyId, fn)
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				return Model{}, err
			}
			err = character.JoinParty(l)(ctx)(characterId, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to add member [%d].", partyId, characterId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] joined party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(joinedEventProvider(characterId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was created.", c.Id())
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Expel(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party.", characterId)
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				return Model{}, err
			}

			p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
				return Model{}, err
			}
			err = character.LeaveParty(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] expelled from party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(expelEventProvider(actorId, p.Id(), c.WorldId(), characterId))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was left.", c.Id())
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
		return func(partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party.", characterId)
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				return Model{}, err
			}

			var disbandParty = p.LeaderId() == characterId

			p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
				return Model{}, err
			}
			err = character.LeaveParty(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
				}
				return Model{}, err
			}

			if disbandParty {
				for _, m := range p.Members() {
					err = character.LeaveParty(l)(ctx)(m)
					if err != nil {
						l.WithError(err).Errorf("Character [%d] stuck in party [%d].", m, partyId)
					}
				}

				GetRegistry().Remove(t, partyId)
				l.Debugf("Party [%d] has been disbanded.", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(disbandEventProvider(characterId, p.Id(), c.WorldId(), p.Members()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] was disbanded.", c.Id())
					return Model{}, err
				}
			} else {
				l.Debugf("Character [%d] left party [%d].", characterId, partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(leftEventProvider(characterId, p.Id(), c.WorldId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] was left.", c.Id())
					return Model{}, err
				}
			}

			return p, nil
		}
	}
}

func ChangeLeader(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party. Cannot become leader.", characterId)
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Update(t, partyId, func(m Model) Model { return Model.SetLeader(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				return Model{}, err
			}

			l.Debugf("Character [%d] became leader of party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(changeLeaderEventProvider(actorId, p.Id(), c.WorldId(), characterId))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce leadership change in party [%d].", c.Id())
				return Model{}, err
			}
			return p, nil
		}
	}
}
