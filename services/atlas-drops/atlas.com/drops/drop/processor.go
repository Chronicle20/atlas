package drop

import (
	"atlas-drops/equipment"
	"atlas-drops/kafka/message"
	"atlas-drops/kafka/message/drop"
	"atlas-drops/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for drop processing operations
type Processor interface {
	// Spawn creates a new drop
	Spawn(mb *message.Buffer) func(mb *ModelBuilder) (Model, error)
	// SpawnAndEmit creates a new drop and emits a Kafka message
	SpawnAndEmit(mb *ModelBuilder) (Model, error)

	// SpawnForCharacter creates a new drop for a character
	SpawnForCharacter(mb *message.Buffer) func(mb *ModelBuilder) (Model, error)
	// SpawnForCharacterAndEmit creates a new drop for a character and emits a Kafka message
	SpawnForCharacterAndEmit(mb *ModelBuilder) (Model, error)

	// Reserve reserves a drop for a character
	Reserve(mb *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) (Model, error)
	// ReserveAndEmit reserves a drop for a character and emits a Kafka message
	ReserveAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) (Model, error)

	// CancelReservation cancels a drop reservation
	CancelReservation(mb *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error
	// CancelReservationAndEmit cancels a drop reservation and emits a Kafka message
	CancelReservationAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error

	// Gather gathers a drop
	Gather(mb *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) (Model, error)
	// GatherAndEmit gathers a drop and emits a Kafka message
	GatherAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) (Model, error)

	// Expire expires a drop
	Expire(mb *message.Buffer) model.Operator[Model]
	// ExpireAndEmit expires a drop and emits a Kafka message
	ExpireAndEmit(m Model) error

	// GetById gets a drop by ID
	GetById(dropId uint32) (Model, error)
	// GetForMap gets all drops for a map
	GetForMap(worldId byte, channelId byte, mapId uint32) ([]Model, error)

	// ByIdProvider provides a drop by ID
	ByIdProvider(dropId uint32) model.Provider[Model]
	// ForMapProvider provides all drops for a map
	ForMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new drop processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// Spawn creates a new drop
func (p *ProcessorImpl) Spawn(msgBuf *message.Buffer) func(mb *ModelBuilder) (Model, error) {
	return func(mb *ModelBuilder) (Model, error) {
		it, _ := inventory.TypeFromItemId(item.Id(mb.ItemId()))
		if it == inventory.TypeValueEquip {
			e, err := equipment.Create(p.l)(p.ctx)(mb.ItemId())()
			if err != nil {
				p.l.WithError(err).Errorf("Unable to generate [%d] equipment for drop.", mb.ItemId())
				return Model{}, err
			}

			mb.SetEquipmentId(e.Id())
		}
		m := GetRegistry().CreateDrop(mb)
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
		m := GetRegistry().CreateDrop(mb)
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
func (p *ProcessorImpl) Reserve(msgBuf *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) (Model, error) {
	return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) (Model, error) {
		d, err := GetRegistry().ReserveDrop(dropId, characterId, petSlot)
		if err == nil {
			p.l.Debugf("Reserving [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservedEventStatusProvider(worldId, channelId, mapId, dropId, characterId, d.ItemId(), d.EquipmentId(), d.Quantity(), d.Meso()))
		} else {
			p.l.Debugf("Failed reserving [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservationFailureEventStatusProvider(worldId, channelId, mapId, dropId, characterId))
		}
		return d, err
	}
}

// ReserveAndEmit reserves a drop for a character and emits a Kafka message
func (p *ProcessorImpl) ReserveAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(mb *message.Buffer) error {
		result, err = p.Reserve(mb)(worldId, channelId, mapId, dropId, characterId, petSlot)
		return err
	})
	return result, err
}

// CancelReservation cancels a drop reservation
func (p *ProcessorImpl) CancelReservation(msgBuf *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
		GetRegistry().CancelDropReservation(dropId, characterId)
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservationFailureEventStatusProvider(worldId, channelId, mapId, dropId, characterId))
		return nil
	}
}

// CancelReservationAndEmit cancels a drop reservation and emits a Kafka message
func (p *ProcessorImpl) CancelReservationAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	err := message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.CancelReservation(mb)(worldId, channelId, mapId, dropId, characterId)
	})
	return err
}

// Gather gathers a drop
func (p *ProcessorImpl) Gather(msgBuf *message.Buffer) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) (Model, error) {
	return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) (Model, error) {
		d, err := GetRegistry().RemoveDrop(dropId)
		if d.Id() == 0 || err == nil {
			p.l.Debugf("Gathering [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, pickedUpEventStatusProvider(worldId, channelId, mapId, dropId, characterId, d.ItemId(), d.EquipmentId(), d.Quantity(), d.Meso(), d.PetSlot()))
		}
		return d, err
	}
}

// GatherAndEmit gathers a drop and emits a Kafka message
func (p *ProcessorImpl) GatherAndEmit(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) (Model, error) {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	var result Model
	var err error
	err = message.Emit(producerProvider)(func(mb *message.Buffer) error {
		result, err = p.Gather(mb)(worldId, channelId, mapId, dropId, characterId)
		return err
	})
	return result, err
}

// Expire expires a drop
func (p *ProcessorImpl) Expire(msgBuf *message.Buffer) model.Operator[Model] {
	return func(m Model) error {
		_, err := GetRegistry().RemoveDrop(m.Id())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to remove drop [%d] from registry.", m.Id())
			return err
		}

		if m.EquipmentId() != 0 {
			err = equipment.Delete(p.l)(p.ctx)(m.EquipmentId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete equipment [%d] corresponding to drop [%d].", m.EquipmentId(), m.Id())
				return err
			}
		}

		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, expiredEventStatusProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.Id()))
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
func (p *ProcessorImpl) GetForMap(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return model.SliceMap[Model, Model](func(m Model) (Model, error) { return m, nil })(p.ForMapProvider(worldId, channelId, mapId))(model.ParallelMap())()
}

// ByIdProvider provides a drop by ID
func (p *ProcessorImpl) ByIdProvider(dropId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetRegistry().GetDrop(dropId)
	}
}

// ForMapProvider provides all drops for a map
func (p *ProcessorImpl) ForMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetRegistry().GetDropsForMap(p.t, worldId, channelId, mapId)
	}
}

// AllProvider provides all drops
var AllProvider = func() ([]Model, error) {
	return GetRegistry().GetAllDrops(), nil
}
