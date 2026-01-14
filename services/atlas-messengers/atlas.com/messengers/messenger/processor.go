package messenger

import (
	"atlas-messengers/character"
	"atlas-messengers/invite"
	"atlas-messengers/kafka/message"
	"atlas-messengers/kafka/message/messenger"
	"atlas-messengers/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
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
	return GetAllProvider(ctx)
}

func byIdProvider(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
	return ByIdProvider(ctx)
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, mm := range m.Members() {
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

func Create(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) (Model, error) {
		return func(transactionID uuid.UUID, characterId uint32) (Model, error) {
			createAndJoinLock.Lock()
			defer createAndJoinLock.Unlock()

			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger create error to [%d].", characterId)
				}
				return Model{}, err
			}

			if c.MessengerId() != 0 {
				l.Errorf("Character [%d] already in messenger. Cannot create another one.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined1, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger create error to [%d].", characterId)
				}
				return Model{}, ErrAlreadyIn
			}

			p := CreateMessenger(ctx)(characterId)

			l.Debugf("Created messenger [%d] for character [%d].", p.Id(), characterId)

			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(createdEventProvider(transactionID, characterId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was created.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return Model{}, err
			}

			err = character.JoinMessenger(l)(ctx)(transactionID, characterId, p.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to have character [%d] join messenger [%d]", characterId, p.Id())
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Join(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
		return func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if c.MessengerId() != 0 {
				l.Errorf("Character [%d] already in messenger. Cannot create another one.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrAlreadyIn
			}

			p, err := ByIdProvider(ctx)(messengerId)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if len(p.Members()) >= 3 {
				l.Errorf("Messenger [%d] already at capacity.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrAtCapacity
			}

			fn := func(m Model) Model { return Model.AddMember(m, characterId) }
			p, err = UpdateMessenger(ctx)(messengerId, fn)
			if err != nil {
				l.WithError(err).Errorf("Unable to join messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			err = character.JoinMessenger(l)(ctx)(transactionID, characterId, messengerId)
			if err != nil {
				l.WithError(err).Errorf("Unable to join messenger [%d].", messengerId)
				p, err = UpdateMessenger(ctx)(messengerId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up messenger [%d], when failing to add member [%d].", messengerId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] joined messenger [%d].", characterId, messengerId)
			mm, _ := p.FindMember(characterId)
			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(joinedEventProvider(transactionID, characterId, p.Id(), c.WorldId(), mm.Slot()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was created.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
		return func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			if c.MessengerId() != messengerId {
				l.Errorf("Character [%d] not in messenger.", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, ErrNotIn
			}

			p, err := ByIdProvider(ctx)(messengerId)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			mm, _ := p.FindMember(characterId)

			p, err = UpdateMessenger(ctx)(messengerId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to leave messenger [%d].", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}
			err = character.LeaveMessenger(l)(ctx)(transactionID, characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to leave messenger [%d].", messengerId)
				p, err = UpdateMessenger(ctx)(messengerId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up messenger [%d], when failing to remove member [%d].", messengerId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", characterId)
				}
				return Model{}, err
			}

			if len(p.Members()) == 0 {
				DeleteMessenger(ctx)(messengerId)
				l.Debugf("Messenger [%d] has been disbanded.", messengerId)
			}

			l.Debugf("Character [%d] left messenger [%d].", characterId, messengerId)
			err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(leftEventProvider(transactionID, characterId, messengerId, c.WorldId(), mm.Slot()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the messenger [%d] was left.", messengerId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, characterId, messengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", messengerId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, actorId uint32, characterId uint32) error {
	return func(ctx context.Context) func(transactionID uuid.UUID, actorId uint32, characterId uint32) error {
		return func(transactionID uuid.UUID, actorId uint32, characterId uint32) error {
			createAndJoinLock.Lock()
			defer createAndJoinLock.Unlock()

			a, err := character.GetById(l)(ctx)(actorId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", actorId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, actorId, 0, a.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", 0)
				}
				return err
			}

			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, actorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return err
			}

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, actorId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, c.Name()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return ErrAlreadyIn
			}

			var p Model
			if a.MessengerId() == 0 {
				p, err = Create(l)(ctx)(transactionID, actorId)
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
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, actorId, p.Id(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", p.Id())
				}
				return ErrAtCapacity
			}

			err = invite.Create(l)(ctx)(transactionID, actorId, a.WorldId(), p.Id(), characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce messenger [%d] invite.", p.Id())
				err = producer.ProviderImpl(l)(ctx)(messenger.EnvEventStatusTopic)(errorEventProvider(transactionID, actorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] error.", a.MessengerId())
				}
				return err
			}

			return nil
		}
	}
}

// ============================================================================
// ProcessorImpl - struct-based processor pattern for consistency with other services
// ============================================================================

// ProcessorImpl provides struct-based processor methods for messenger operations.
// This pattern is consistent with other Atlas services and allows for easier testing.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new ProcessorImpl with the given logger and context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// Create creates a new messenger for the given character.
// Note: This method includes inline Kafka emissions. Use CreateAndEmit for buffered emissions.
func (p *ProcessorImpl) Create(transactionID uuid.UUID, characterId uint32) (Model, error) {
	return Create(p.l)(p.ctx)(transactionID, characterId)
}

// CreateAndEmit creates a new messenger and emits events via buffer pattern.
func (p *ProcessorImpl) CreateAndEmit(input CreateInput) (Model, error) {
	return CreateAndEmit(p.l)(p.ctx)(input)
}

// Join adds a character to an existing messenger.
// Note: This method includes inline Kafka emissions. Use JoinAndEmit for buffered emissions.
func (p *ProcessorImpl) Join(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
	return Join(p.l)(p.ctx)(transactionID, messengerId, characterId)
}

// JoinAndEmit adds a character to a messenger and emits events via buffer pattern.
func (p *ProcessorImpl) JoinAndEmit(input JoinInput) (Model, error) {
	return JoinAndEmit(p.l)(p.ctx)(input)
}

// Leave removes a character from a messenger.
// Note: This method includes inline Kafka emissions. Use LeaveAndEmit for buffered emissions.
func (p *ProcessorImpl) Leave(transactionID uuid.UUID, messengerId uint32, characterId uint32) (Model, error) {
	return Leave(p.l)(p.ctx)(transactionID, messengerId, characterId)
}

// LeaveAndEmit removes a character from a messenger and emits events via buffer pattern.
func (p *ProcessorImpl) LeaveAndEmit(input LeaveInput) (Model, error) {
	return LeaveAndEmit(p.l)(p.ctx)(input)
}

// RequestInvite sends an invitation to another character to join a messenger.
// Note: This method includes inline Kafka emissions. Use RequestInviteAndEmit for buffered emissions.
func (p *ProcessorImpl) RequestInvite(transactionID uuid.UUID, actorId uint32, characterId uint32) error {
	return RequestInvite(p.l)(p.ctx)(transactionID, actorId, characterId)
}

// RequestInviteAndEmit sends an invitation and emits events via buffer pattern.
func (p *ProcessorImpl) RequestInviteAndEmit(input RequestInviteInput) error {
	return RequestInviteAndEmit(p.l)(p.ctx)(input)
}

// GetById retrieves a messenger by its ID.
func (p *ProcessorImpl) GetById(messengerId uint32) (Model, error) {
	return GetById(p.ctx)(messengerId)
}

// GetSlice retrieves messengers matching the given filters.
func (p *ProcessorImpl) GetSlice(filters ...model.Filter[Model]) ([]Model, error) {
	return GetSlice(p.ctx)(filters...)
}

// ============================================================================
// AndEmit variants - separate business logic from event emission using buffer
// ============================================================================

// CreateInput holds the input parameters for CreateAndEmit
type CreateInput struct {
	TransactionID uuid.UUID
	CharacterId   uint32
}

// CreateAndEmit creates a messenger and emits all events via buffer pattern.
// Events are emitted regardless of success or failure.
func CreateAndEmit(l logrus.FieldLogger) func(ctx context.Context) func(input CreateInput) (Model, error) {
	return func(ctx context.Context) func(input CreateInput) (Model, error) {
		ep := producer.ProviderImpl(l)(ctx)
		return message.EmitAlways[Model, CreateInput](ep)(func(buf *message.Buffer) func(CreateInput) (Model, error) {
			return func(input CreateInput) (Model, error) {
				createAndJoinLock.Lock()
				defer createAndJoinLock.Unlock()

				c, err := character.GetById(l)(ctx)(input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Error getting character [%d].", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				if c.MessengerId() != 0 {
					l.Errorf("Character [%d] already in messenger. Cannot create another one.", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined1, ""))
					return Model{}, ErrAlreadyIn
				}

				p := CreateMessenger(ctx)(input.CharacterId)
				l.Debugf("Created messenger [%d] for character [%d].", p.Id(), input.CharacterId)

				err = buf.Put(messenger.EnvEventStatusTopic, createdEventProvider(input.TransactionID, input.CharacterId, p.Id(), c.WorldId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to buffer messenger created event.")
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				err = character.JoinMessenger(l)(ctx)(input.TransactionID, input.CharacterId, p.Id())
				if err != nil {
					l.WithError(err).Errorf("Unable to have character [%d] join messenger [%d]", input.CharacterId, p.Id())
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, 0, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				return p, nil
			}
		})
	}
}

// JoinInput holds the input parameters for JoinAndEmit
type JoinInput struct {
	TransactionID uuid.UUID
	MessengerId   uint32
	CharacterId   uint32
}

// JoinAndEmit joins a character to a messenger and emits all events via buffer pattern.
func JoinAndEmit(l logrus.FieldLogger) func(ctx context.Context) func(input JoinInput) (Model, error) {
	return func(ctx context.Context) func(input JoinInput) (Model, error) {
		ep := producer.ProviderImpl(l)(ctx)
		return message.EmitAlways[Model, JoinInput](ep)(func(buf *message.Buffer) func(JoinInput) (Model, error) {
			return func(input JoinInput) (Model, error) {
				c, err := character.GetById(l)(ctx)(input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Error getting character [%d].", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				if c.MessengerId() != 0 {
					l.Errorf("Character [%d] already in messenger. Cannot join another one.", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, ""))
					return Model{}, ErrAlreadyIn
				}

				p, err := ByIdProvider(ctx)(input.MessengerId)()
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve messenger [%d].", input.MessengerId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				if len(p.Members()) >= MaxMembers {
					l.Errorf("Messenger [%d] already at capacity.", input.MessengerId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
					return Model{}, ErrAtCapacity
				}

				fn := func(m Model) Model { return Model.AddMember(m, input.CharacterId) }
				p, err = UpdateMessenger(ctx)(input.MessengerId, fn)
				if err != nil {
					l.WithError(err).Errorf("Unable to join messenger [%d].", input.MessengerId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				err = character.JoinMessenger(l)(ctx)(input.TransactionID, input.CharacterId, input.MessengerId)
				if err != nil {
					l.WithError(err).Errorf("Unable to join messenger [%d].", input.MessengerId)
					// Rollback messenger update
					_, _ = UpdateMessenger(ctx)(input.MessengerId, func(m Model) Model { return Model.RemoveMember(m, input.CharacterId) })
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				l.Debugf("Character [%d] joined messenger [%d].", input.CharacterId, input.MessengerId)
				mm, _ := p.FindMember(input.CharacterId)
				_ = buf.Put(messenger.EnvEventStatusTopic, joinedEventProvider(input.TransactionID, input.CharacterId, p.Id(), c.WorldId(), mm.Slot()))

				return p, nil
			}
		})
	}
}

// LeaveInput holds the input parameters for LeaveAndEmit
type LeaveInput struct {
	TransactionID uuid.UUID
	MessengerId   uint32
	CharacterId   uint32
}

// LeaveAndEmit removes a character from a messenger and emits all events via buffer pattern.
func LeaveAndEmit(l logrus.FieldLogger) func(ctx context.Context) func(input LeaveInput) (Model, error) {
	return func(ctx context.Context) func(input LeaveInput) (Model, error) {
		ep := producer.ProviderImpl(l)(ctx)
		return message.EmitAlways[Model, LeaveInput](ep)(func(buf *message.Buffer) func(LeaveInput) (Model, error) {
			return func(input LeaveInput) (Model, error) {
				c, err := character.GetById(l)(ctx)(input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Error getting character [%d].", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				if c.MessengerId() != input.MessengerId {
					l.Errorf("Character [%d] not in messenger.", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, ErrNotIn
				}

				p, err := ByIdProvider(ctx)(input.MessengerId)()
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve messenger [%d].", input.MessengerId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}
				mm, _ := p.FindMember(input.CharacterId)

				p, err = UpdateMessenger(ctx)(input.MessengerId, func(m Model) Model { return Model.RemoveMember(m, input.CharacterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to leave messenger [%d].", input.MessengerId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				err = character.LeaveMessenger(l)(ctx)(input.TransactionID, input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to leave messenger [%d].", input.MessengerId)
					// Rollback messenger update
					_, _ = UpdateMessenger(ctx)(input.MessengerId, func(m Model) Model { return Model.AddMember(m, input.CharacterId) })
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return Model{}, err
				}

				if len(p.Members()) == 0 {
					DeleteMessenger(ctx)(input.MessengerId)
					l.Debugf("Messenger [%d] has been disbanded.", input.MessengerId)
				}

				l.Debugf("Character [%d] left messenger [%d].", input.CharacterId, input.MessengerId)
				_ = buf.Put(messenger.EnvEventStatusTopic, leftEventProvider(input.TransactionID, input.CharacterId, input.MessengerId, c.WorldId(), mm.Slot()))

				return p, nil
			}
		})
	}
}

// RequestInviteInput holds the input parameters for RequestInviteAndEmit
type RequestInviteInput struct {
	TransactionID uuid.UUID
	ActorId       uint32
	CharacterId   uint32
}

// RequestInviteAndEmit requests an invitation and emits all events via buffer pattern.
func RequestInviteAndEmit(l logrus.FieldLogger) func(ctx context.Context) func(input RequestInviteInput) error {
	return func(ctx context.Context) func(input RequestInviteInput) error {
		ep := producer.ProviderImpl(l)(ctx)
		return message.EmitAlwaysNoResult[RequestInviteInput](ep)(func(buf *message.Buffer) func(RequestInviteInput) error {
			return func(input RequestInviteInput) error {
				createAndJoinLock.Lock()
				defer createAndJoinLock.Unlock()

				a, err := character.GetById(l)(ctx)(input.ActorId)
				if err != nil {
					l.WithError(err).Errorf("Error getting character [%d].", input.ActorId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.ActorId, 0, a.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return err
				}

				c, err := character.GetById(l)(ctx)(input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Error getting character [%d].", input.CharacterId)
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.ActorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return err
				}

				if c.MessengerId() != 0 {
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.ActorId, c.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAlreadyJoined2, c.Name()))
					return ErrAlreadyIn
				}

				var p Model
				if a.MessengerId() == 0 {
					// Use the AndEmit version for nested create
					p, err = CreateAndEmit(l)(ctx)(CreateInput{TransactionID: input.TransactionID, CharacterId: input.ActorId})
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

				if len(p.Members()) >= MaxMembers {
					l.Errorf("Messenger [%d] already at capacity.", p.Id())
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.ActorId, p.Id(), c.WorldId(), messenger.EventMessengerStatusErrorTypeAtCapacity, ""))
					return ErrAtCapacity
				}

				err = invite.Create(l)(ctx)(input.TransactionID, input.ActorId, a.WorldId(), p.Id(), input.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce messenger [%d] invite.", p.Id())
					_ = buf.Put(messenger.EnvEventStatusTopic, errorEventProvider(input.TransactionID, input.ActorId, a.MessengerId(), c.WorldId(), messenger.EventMessengerStatusErrorUnexpected, ""))
					return err
				}

				return nil
			}
		})
	}
}
