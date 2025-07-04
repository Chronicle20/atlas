package messenger

import (
	"atlas-messengers/character"
	"atlas-messengers/invite"
	"atlas-messengers/kafka/message/messenger"
	"atlas-messengers/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"sync"
)

const StartMessengerId = uint32(1000000000)

var ErrNotFound = errors.New("not found")
var ErrAtCapacity = errors.New("at capacity")
var ErrAlreadyIn = errors.New("already in messenger")
var ErrNotIn = errors.New("not in messenger")
var ErrNotAsBeginner = errors.New("not as beginner")
var ErrNotAsGm = errors.New("not as gm")

func AllProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().GetAll(t), nil
	}
}

func byIdProvider(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
	return func(messengerId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetRegistry().Get(t, messengerId)
		}
	}
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, mm := range m.members {
			if mm.Id() == memberId {
				return true
			}
		}
		return false
	}
}

func GetSlice(ctx context.Context) func(filters ...model.Filter[Model]) ([]Model, error) {
	return func(filters ...model.Filter[Model]) ([]Model, error) {
		return model.FilteredProvider(AllProvider(ctx), model.Filters[Model](filters...))()
	}
}

func GetById(ctx context.Context) func(messengerId uint32) (Model, error) {
	return func(messengerId uint32) (Model, error) {
		return byIdProvider(ctx)(messengerId)()
	}
}

var createAndJoinLock = sync.RWMutex{}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			createAndJoinLock.Lock()
			defer createAndJoinLock.Unlock()

			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger create error to [%d].", characterId)
				}
				return Model{}, err
			}

			if c.MessengerId() != 0 {
				l.Errorf("Character [%d] already in messenger. Cannot create another one.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined1, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger create error to [%d].", characterId)
				}
				return Model{}, ErrAlreadyIn
			}

			p := GetRegistry().Create(t, characterId)

			l.Debugf("Created messenger [%d] for character [%d].", p.Id(), characterId)

			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(createdEventProvider(characterId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was created.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return Model{}, err
			}

			err = character.JoinMessenger(l)(ctx)(characterId, p.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to have character [%d] join messenger [%d]", characterId, p.Id())
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Join(l logrus.FieldLogger) func(ctx context.Context) func(messengerId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(messengerId uint32, characterId uint32) (Model, error) {
		return func(messengerId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if c.MessengerId() != 0 {
				l.Errorf("Character [%d] already in messenger. Cannot create another one.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrAlreadyIn
			}

			p, err := GetRegistry().Get(t, messengerId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if len(p.Members()) >= 3 {
				l.Errorf("Messenger [%d] already at capacity.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrAtCapacity
			}

			fn := func(m Model) Model { return Model.AddMember(m, characterId) }
			p, err = GetRegistry().Update(t, messengerId, fn)
			if err != nil {
				l.WithError(err).Errorf("Unable to join messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			err = character.JoinMessenger(l)(ctx)(characterId, messengerId)
			if err != nil {
				l.WithError(err).Errorf("Unable to join messenger [%d].", messengerId)
				p, err = GetRegistry().Update(t, messengerId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up messenger [%d], when failing to add member [%d].", messengerId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] joined messenger [%d].", characterId, messengerId)
			mm, _ := p.FindMember(characterId)
			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(joinedEventProvider(characterId, p.Id(), c.WorldId(), mm.Slot()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was created.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(messengerId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(messengerId uint32, characterId uint32) (Model, error) {
		return func(messengerId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if c.MessengerId() != messengerId {
				l.Errorf("Character [%d] not in messenger.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Get(t, messengerId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			mm, _ := p.FindMember(characterId)

			p, err = GetRegistry().Update(t, messengerId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to leave messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			err = character.LeaveMessenger(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to leave messenger [%d].", messengerId)
				p, err = GetRegistry().Update(t, messengerId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up messenger [%d], when failing to remove member [%d].", messengerId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", characterId)
				}
				return Model{}, err
			}

			if len(p.Members()) == 0 {
				GetRegistry().Remove(t, messengerId)
				l.Debugf("Messenger [%d] has been disbanded.", messengerId)
			}

			l.Debugf("Character [%d] left messenger [%d].", characterId, messengerId)
			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(leftEventProvider(characterId, messengerId, c.WorldId(), mm.Slot()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was left.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, characterId uint32) error {
	return func(ctx context.Context) func(actorId uint32, characterId uint32) error {
		return func(actorId uint32, characterId uint32) error {
			createAndJoinLock.Lock()
			defer createAndJoinLock.Unlock()

			a, err := character.GetById(l)(ctx)(actorId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", actorId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(actorId, 0, a.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", 0)
				}
				return err
			}

			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(actorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return err
			}

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(actorId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, c.Name()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return ErrAlreadyIn
			}

			var p Model
			if a.MessengerId() == 0 {
				p, err = Create(l)(ctx)(actorId)
				if err != nil {
					l.WithError(err).Errorf("Unable to automatically create messenger [%d].", a.MessengerId())
					return err
				}
			} else {
				p, err = GetById(ctx)(a.MessengerId())
				if err != nil {
					return err
				}
			}

			if len(p.Members()) >= 3 {
				l.Errorf("Messenger [%d] already at capacity.", p.Id())
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(actorId, p.Id(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return ErrAtCapacity
			}

			err = invite.Create(l)(ctx)(actorId, a.WorldId(), p.Id(), characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce messenger [%d] invite.", p.Id())
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(actorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return err
			}

			return nil
		}
	}
}
