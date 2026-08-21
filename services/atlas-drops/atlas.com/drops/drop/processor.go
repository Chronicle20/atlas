package drop

import (
	"atlas-drops/kafka/message"
	"atlas-drops/kafka/message/drop"
	"atlas-drops/party"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor defines the interface for drop processing operations
type Processor interface {
	// With returns a clone of the processor with the given options applied.
	With(opts ...ProcessorOption) Processor

	// Spawn creates a new drop
	Spawn(mb *message.Buffer) func(mb *ModelBuilder) (Model, error)
	// SpawnAndEmit creates a new drop and emits a Kafka message
	SpawnAndEmit(mb *ModelBuilder) (Model, error)

	// SpawnForCharacter creates a new drop for a character
	SpawnForCharacter(mb *message.Buffer) func(mb *ModelBuilder) (Model, error)
	// SpawnForCharacterAndEmit creates a new drop for a character and emits a Kafka message
	SpawnForCharacterAndEmit(mb *ModelBuilder) (Model, error)

	// Reserve reserves a drop for a character
	Reserve(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error)
	// ReserveAndEmit reserves a drop for a character and emits a Kafka message
	ReserveAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error)

	// CancelReservation cancels a drop reservation
	CancelReservation(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error
	// CancelReservationAndEmit cancels a drop reservation and emits a Kafka message
	CancelReservationAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error

	// Gather gathers a drop
	Gather(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (Model, error)
	// GatherAndEmit gathers a drop and emits a Kafka message
	GatherAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (Model, error)

	// Consume removes a drop consumed by a game mechanic (e.g., item-reactor trigger)
	Consume(mb *message.Buffer) func(field field.Model, dropId uint32) error
	// ConsumeAndEmit removes a drop consumed by a game mechanic and emits a Kafka message
	ConsumeAndEmit(field field.Model, dropId uint32) error

	// Expire expires a drop
	Expire(mb *message.Buffer) model.Operator[Model]
	// ExpireAndEmit expires a drop and emits a Kafka message
	ExpireAndEmit(m Model) error

	// GetById gets a drop by ID
	GetById(dropId uint32) (Model, error)
	// GetForMap gets all drops for a map
	GetForMap(f field.Model) ([]Model, error)

	// ByIdProvider provides a drop by ID
	ByIdProvider(dropId uint32) model.Provider[Model]
	// ForMapProvider provides all drops for a map
	ForMapProvider(f field.Model) model.Provider[[]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	pp  party.Processor
}

// NewProcessor creates a new drop processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		pp:  party.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// ProcessorOption customizes a ProcessorImpl clone produced by With.
type ProcessorOption func(*ProcessorImpl)

// WithPartyProcessor overrides the atlas-parties client. Tests inject a mock;
// production always uses the default from NewProcessor.
func WithPartyProcessor(pp party.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.pp = pp
	}
}

// With returns a clone of the processor with the given options applied.
func (p *ProcessorImpl) With(opts ...ProcessorOption) Processor {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}

// Spawn creates a new drop (equipment stats already inline from command)
func (p *ProcessorImpl) Spawn(msgBuf *message.Buffer) func(mb *ModelBuilder) (Model, error) {
	return func(mb *ModelBuilder) (Model, error) {
		m, err := GetRegistry().CreateDrop(mb)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create drop.")
			return Model{}, err
		}
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, createdEventStatusProvider(m))
		return m, nil
	}
}

// SpawnAndEmit creates a new drop and emits a Kafka message
func (p *ProcessorImpl) SpawnAndEmit(mb *ModelBuilder) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(msgBuf *message.Buffer) error {
		result, err = p.Spawn(msgBuf)(mb)
		return err
	})
	return result, err
}

// SpawnForCharacter creates a new drop for a character
func (p *ProcessorImpl) SpawnForCharacter(msgBuf *message.Buffer) func(mb *ModelBuilder) (Model, error) {
	return func(mb *ModelBuilder) (Model, error) {
		m, err := GetRegistry().CreateDrop(mb)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create drop for character.")
			return Model{}, err
		}
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, createdEventStatusProvider(m))
		return m, nil
	}
}

// SpawnForCharacterAndEmit creates a new drop for a character and emits a Kafka message
func (p *ProcessorImpl) SpawnForCharacterAndEmit(mb *ModelBuilder) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(msgBuf *message.Buffer) error {
		result, err = p.SpawnForCharacter(msgBuf)(mb)
		return err
	})
	return result, err
}

// Reserve reserves a drop for a character
func (p *ProcessorImpl) Reserve(msgBuf *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
	return func(transactionId uuid.UUID, f field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
		d, err := GetRegistry().ReserveDrop(p.t, dropId, characterId, partyId, petSlot)
		if err != nil {
			p.l.Debugf("Failed reserving [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservationFailureEventStatusProvider(transactionId, f, dropId, characterId))
			return d, err
		}
		p.l.Debugf("Reserving [%d] for [%d].", dropId, characterId)
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservedEventStatusProvider(transactionId, f, d, characterId))
		if d.Meso() == 0 {
			return d, nil
		}
		rs := splitMeso(f, d.Meso(), characterId, p.resolveMembers(characterId))
		p.l.Debugf("Splitting [%d] meso from drop [%d] among [%d] recipient(s).", d.Meso(), dropId, len(rs))
		for _, r := range rs {
			// A zero share is suppressed for everyone but the picker: the
			// picker's award is what completes the pickup, so it must be
			// emitted even at Amount 0 or the drop never leaves the map.
			if r.Amount == 0 && !r.Picker {
				continue
			}
			p.l.Debugf("Awarding [%d] meso from drop [%d] to character [%d].", r.Amount, dropId, r.CharacterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, mesoAwardedEventStatusProvider(transactionId, f, dropId, r))
		}
		return d, nil
	}
}

// resolveMembers returns the picker's party roster, or nil when the lookup
// fails. It deliberately returns no error: an atlas-parties outage degrades to
// a full-amount award to the picker and must never fail the pickup, and this
// signature makes that impossible to get wrong at a call site.
func (p *ProcessorImpl) resolveMembers(characterId uint32) []party.MemberModel {
	m, err := p.pp.GetByMemberId(characterId)
	if err != nil {
		degrade.Observe(p.l, "drops.meso_split.party", characterId, err)
		return nil
	}
	return m.Members()
}

// ReserveAndEmit reserves a drop for a character and emits a Kafka message
func (p *ProcessorImpl) ReserveAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(mb *message.Buffer) error {
		result, err = p.Reserve(mb)(transactionId, field, dropId, characterId, partyId, petSlot)
		return err
	})
	return result, err
}

// CancelReservation cancels a drop reservation
func (p *ProcessorImpl) CancelReservation(msgBuf *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
	return func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
		_, err := GetRegistry().GetDrop(p.t, dropId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to cancel reservation for [%d].", dropId)
		}
		GetRegistry().CancelDropReservation(p.t, dropId, characterId)
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservationFailureEventStatusProvider(transactionId, field, dropId, characterId))
		return nil
	}
}

// CancelReservationAndEmit cancels a drop reservation and emits a Kafka message
func (p *ProcessorImpl) CancelReservationAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	err := message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.CancelReservation(mb)(transactionId, field, dropId, characterId)
	})
	return err
}

// Gather gathers a drop
func (p *ProcessorImpl) Gather(msgBuf *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (Model, error) {
	return func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (Model, error) {
		d, err := GetRegistry().RemoveDrop(p.t, dropId)
		if d.Id() == 0 || err == nil {
			p.l.Debugf("Gathering [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, pickedUpEventStatusProvider(transactionId, field, d, characterId))
		}
		return d, err
	}
}

// GatherAndEmit gathers a drop and emits a Kafka message
func (p *ProcessorImpl) GatherAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(mb *message.Buffer) error {
		result, err = p.Gather(mb)(transactionId, field, dropId, characterId)
		return err
	})
	return result, err
}

// Consume removes a drop consumed by a game mechanic
func (p *ProcessorImpl) Consume(msgBuf *message.Buffer) func(field field.Model, dropId uint32) error {
	return func(field field.Model, dropId uint32) error {
		d, err := GetRegistry().RemoveDrop(p.t, dropId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to consume drop [%d].", dropId)
			return err
		}
		if d.Id() == 0 {
			return nil
		}
		p.l.Debugf("Consuming drop [%d].", dropId)
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, consumedEventStatusProvider(d.TransactionId(), field, dropId))
		return nil
	}
}

// ConsumeAndEmit removes a drop consumed by a game mechanic and emits a Kafka message
func (p *ProcessorImpl) ConsumeAndEmit(field field.Model, dropId uint32) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	return message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.Consume(mb)(field, dropId)
	})
}

// Expire expires a drop
func (p *ProcessorImpl) Expire(msgBuf *message.Buffer) model.Operator[Model] {
	return func(m Model) error {
		_, err := GetRegistry().RemoveDrop(m.Tenant(), m.Id())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to remove drop [%d] from registry.", m.Id())
			return err
		}

		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, expiredEventStatusProvider(m.TransactionId(), m.Field(), m.Id()))
		return nil
	}
}

// ExpireAndEmit expires a drop and emits a Kafka message
func (p *ProcessorImpl) ExpireAndEmit(m Model) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	return message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.Expire(mb)(m)
	})
}

// GetById gets a drop by ID
func (p *ProcessorImpl) GetById(dropId uint32) (Model, error) {
	return model.Map[Model, Model](func(m Model) (Model, error) { return m, nil })(p.ByIdProvider(dropId))()
}

// GetForMap gets all drops for a map
func (p *ProcessorImpl) GetForMap(f field.Model) ([]Model, error) {
	return model.SliceMap[Model, Model](func(m Model) (Model, error) { return m, nil })(p.ForMapProvider(f))(model.ParallelMap())()
}

// ByIdProvider provides a drop by ID
func (p *ProcessorImpl) ByIdProvider(dropId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetRegistry().GetDrop(p.t, dropId)
	}
}

// ForMapProvider provides all drops for a map
func (p *ProcessorImpl) ForMapProvider(f field.Model) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetRegistry().GetDropsForMap(p.t, f)
	}
}

// AllProvider provides all drops
var AllProvider = func() ([]Model, error) {
	return GetRegistry().GetAllDrops(), nil
}
