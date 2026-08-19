package parcel

import (
	"atlas-saga-orchestrator/kafka/message"
	parcelmsg "atlas-saga-orchestrator/kafka/message/parcel"
	parcelCustody "atlas-saga-orchestrator/kafka/message/parcel/custody"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// AcceptToParcelParams carries the full parcel-creation snapshot dispatched to
// atlas-parcel's custody consumer. It mirrors the AcceptToParcelCommandBody
// wire shape (kafka/message/parcel/custody/kafka.go) so atlas-parcel can
// CREATE the parcel row in custody from data alone (the item already left
// inventory). Grouped into a struct because the field count exceeds a
// readable positional argument list. Mirrors AcceptToMtsListingParams.
type AcceptToParcelParams struct {
	ParcelId           uuid.UUID
	CharacterId        uint32
	WorldId            world.Id
	SenderAccountId    uint32
	SenderName         string
	RecipientId        uint32
	RecipientAccountId uint32
	MesoAmount         uint32
	FeePaid            uint32
	Quick              bool
	Message            string
	ReceivableAt       time.Time
	ExpiresAt          time.Time

	HasItem bool

	TemplateId    uint32
	Quantity      uint32
	Strength      uint16
	Dexterity     uint16
	Intelligence  uint16
	Luck          uint16
	HP            uint16
	MP            uint16
	WeaponAttack  uint16
	MagicAttack   uint16
	WeaponDefense uint16
	MagicDefense  uint16
	Accuracy      uint16
	Avoidability  uint16
	Hands         uint16
	Speed         uint16
	Jump          uint16
	Slots         uint16
	Level         byte
	ItemLevel     byte
	ItemExp       uint32
	RingId        uint32
	ViciousCount  uint32
	Flags         uint16
	Owner         string
}

// Processor dispatches the atomic parcel custody commands (AcceptToParcel /
// ReleaseFromParcel / RestoreParcel / RemoveParcel) to atlas-parcel via
// COMMAND_TOPIC_PARCEL_CUSTODY. It mirrors mts.Processor's dispatch to
// COMMAND_TOPIC_MTS_CUSTODY exactly — pure Buffer methods plus AndEmit
// wrappers. RestoreParcel and RemoveParcel exist for the compensator (Task
// 14).
type Processor interface {
	AcceptToParcelAndEmit(transactionId uuid.UUID, params AcceptToParcelParams) error
	AcceptToParcel(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToParcelParams) error
	ReleaseFromParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) error
	ReleaseFromParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) error
	RestoreParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID) error
	RestoreParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID) error
	RemoveParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID) error
	RemoveParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID) error
	// ShowParcelAndEmit and ShowParcel dispatch the SHOW_PARCEL command to
	// atlas-channel via COMMAND_TOPIC_PARCEL, mirroring
	// storage.Processor.ShowStorageAndEmit. Self-completing: the caller marks
	// the step done immediately after the command is sent (Task 19).
	ShowParcelAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) error
	ShowParcel(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) error
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

func (p *ProcessorImpl) AcceptToParcelAndEmit(transactionId uuid.UUID, params AcceptToParcelParams) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AcceptToParcel(mb)(transactionId, params)
	})
}

func (p *ProcessorImpl) AcceptToParcel(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToParcelParams) error {
	return func(transactionId uuid.UUID, params AcceptToParcelParams) error {
		return mb.Put(parcelCustody.EnvCommandTopic, AcceptToParcelProvider(transactionId, params))
	}
}

func (p *ProcessorImpl) ReleaseFromParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ReleaseFromParcel(mb)(transactionId, parcelId, recipientId)
	})
}

func (p *ProcessorImpl) ReleaseFromParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) error {
	return func(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) error {
		return mb.Put(parcelCustody.EnvCommandTopic, ReleaseFromParcelProvider(transactionId, parcelId, recipientId))
	}
}

func (p *ProcessorImpl) RestoreParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RestoreParcel(mb)(transactionId, parcelId)
	})
}

func (p *ProcessorImpl) RestoreParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID) error {
	return func(transactionId uuid.UUID, parcelId uuid.UUID) error {
		return mb.Put(parcelCustody.EnvCommandTopic, RestoreParcelProvider(transactionId, parcelId))
	}
}

func (p *ProcessorImpl) RemoveParcelAndEmit(transactionId uuid.UUID, parcelId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RemoveParcel(mb)(transactionId, parcelId)
	})
}

func (p *ProcessorImpl) RemoveParcel(mb *message.Buffer) func(transactionId uuid.UUID, parcelId uuid.UUID) error {
	return func(transactionId uuid.UUID, parcelId uuid.UUID) error {
		return mb.Put(parcelCustody.EnvCommandTopic, RemoveParcelProvider(transactionId, parcelId))
	}
}

func (p *ProcessorImpl) ShowParcelAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ShowParcel(mb)(transactionId, ch, characterId, npcId, quick)
	})
}

func (p *ProcessorImpl) ShowParcel(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) error {
	return func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) error {
		return mb.Put(parcelmsg.EnvCommandTopic, ShowParcelCommandProvider(transactionId, ch, characterId, npcId, quick))
	}
}
