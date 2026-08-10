package trade

import (
	"atlas-saga-orchestrator/kafka/message"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tradeCustody "atlas-saga-orchestrator/kafka/message/trade/custody"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// AcceptToTradeParams carries the full escrow-creation snapshot dispatched to
// atlas-trades' custody consumer. It mirrors AcceptToTradeCommandBody rather
// than reusing it so the saga layer never depends on the wire struct directly —
// the same separation mts.AcceptToMtsListingParams keeps.
type AcceptToTradeParams struct {
	EscrowId            uuid.UUID
	RoomId              uuid.UUID
	OwnerId             uint32
	TradeSlot           byte
	SourceInventoryType byte
	SourceSlot          int16
	AssetId             uint32

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

// Processor dispatches the atomic trade-escrow custody commands to atlas-trades
// via COMMAND_TOPIC_TRADE_CUSTODY. It mirrors mts.Processor exactly — pure
// Buffer methods plus AndEmit wrappers, so a caller can batch a custody command
// into the same transactional-outbox write as whatever else it is emitting.
type Processor interface {
	AcceptToTradeAndEmit(transactionId uuid.UUID, params AcceptToTradeParams) error
	AcceptToTrade(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToTradeParams) error
	ReleaseFromTradeAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error
	ReleaseFromTrade(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error
	RestoreTradeEscrowAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error
	RestoreTradeEscrow(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error
	RemoveTradeEscrowAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error
	RemoveTradeEscrow(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error
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

func (p *ProcessorImpl) AcceptToTradeAndEmit(transactionId uuid.UUID, params AcceptToTradeParams) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AcceptToTrade(mb)(transactionId, params)
	})
}

func (p *ProcessorImpl) AcceptToTrade(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToTradeParams) error {
	return func(transactionId uuid.UUID, params AcceptToTradeParams) error {
		return mb.Put(tradeCustody.EnvCommandTopic, AcceptToTradeProvider(transactionId, params))
	}
}

func (p *ProcessorImpl) ReleaseFromTradeAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ReleaseFromTrade(mb)(transactionId, escrowId)
	})
}

func (p *ProcessorImpl) ReleaseFromTrade(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return func(transactionId uuid.UUID, escrowId uuid.UUID) error {
		return mb.Put(tradeCustody.EnvCommandTopic, ReleaseFromTradeProvider(transactionId, escrowId))
	}
}

func (p *ProcessorImpl) RestoreTradeEscrowAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RestoreTradeEscrow(mb)(transactionId, escrowId)
	})
}

func (p *ProcessorImpl) RestoreTradeEscrow(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return func(transactionId uuid.UUID, escrowId uuid.UUID) error {
		return mb.Put(tradeCustody.EnvCommandTopic, RestoreTradeEscrowProvider(transactionId, escrowId))
	}
}

func (p *ProcessorImpl) RemoveTradeEscrowAndEmit(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RemoveTradeEscrow(mb)(transactionId, escrowId)
	})
}

func (p *ProcessorImpl) RemoveTradeEscrow(mb *message.Buffer) func(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return func(transactionId uuid.UUID, escrowId uuid.UUID) error {
		return mb.Put(tradeCustody.EnvCommandTopic, RemoveTradeEscrowProvider(transactionId, escrowId))
	}
}
