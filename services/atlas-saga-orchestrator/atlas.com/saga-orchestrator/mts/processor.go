package mts

import (
	"atlas-saga-orchestrator/kafka/message"
	mtsCustody "atlas-saga-orchestrator/kafka/message/mts/custody"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AcceptToMtsListingParams carries the full listing-creation snapshot dispatched
// to atlas-mts's custody consumer. It mirrors the AcceptToMtsListingCommandBody
// wire shape (atlas-mts kafka/message/custody/kafka.go) so atlas-mts can CREATE
// the listing row in active state from data alone (the item already left
// inventory). Grouped into a struct because the field count exceeds a readable
// positional argument list.
type AcceptToMtsListingParams struct {
	ListingId       uuid.UUID
	WorldId         byte
	SellerId        uint32
	SellerAccountId uint32
	SellerName      string
	SaleType        string

	TemplateId uint32
	Quantity   uint32

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

	ListValue      uint32
	BuyNowPrice    *uint32
	CommissionRate float64
	Category       string
	SubCategory    string
	EndsAt         *time.Time
	MinIncrement   uint32

	OfferWishSerial  uint32
	OfferWishOwnerId uint32
}

// Processor dispatches the atomic MTS custody commands (AcceptToMtsListing /
// ReleaseFromMtsHolding / MtsMoveListingToHolding) to atlas-mts via
// COMMAND_TOPIC_MTS_CUSTODY. It mirrors cashshop.Processor's dispatch to
// COMMAND_TOPIC_CASH_COMPARTMENT exactly — pure Buffer methods plus AndEmit
// wrappers.
type Processor interface {
	AcceptToMtsListingAndEmit(transactionId uuid.UUID, params AcceptToMtsListingParams) error
	AcceptToMtsListing(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToMtsListingParams) error
	ReleaseFromMtsHoldingAndEmit(transactionId uuid.UUID, holdingId uuid.UUID) error
	ReleaseFromMtsHolding(mb *message.Buffer) func(transactionId uuid.UUID, holdingId uuid.UUID) error
	RestoreMtsHoldingAndEmit(transactionId uuid.UUID, holdingId uuid.UUID) error
	RestoreMtsHolding(mb *message.Buffer) func(transactionId uuid.UUID, holdingId uuid.UUID) error
	MoveListingToHoldingAndEmit(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) error
	MoveListingToHolding(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) error
	RemoveMtsListingAndEmit(transactionId uuid.UUID, listingId uuid.UUID) error
	RemoveMtsListing(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID) error
	RestoreListingFromHoldingAndEmit(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) error
	RestoreListingFromHolding(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) error
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

func (p *ProcessorImpl) AcceptToMtsListingAndEmit(transactionId uuid.UUID, params AcceptToMtsListingParams) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AcceptToMtsListing(mb)(transactionId, params)
	})
}

func (p *ProcessorImpl) AcceptToMtsListing(mb *message.Buffer) func(transactionId uuid.UUID, params AcceptToMtsListingParams) error {
	return func(transactionId uuid.UUID, params AcceptToMtsListingParams) error {
		return mb.Put(mtsCustody.EnvCommandTopic, AcceptToMtsListingProvider(transactionId, params))
	}
}

func (p *ProcessorImpl) ReleaseFromMtsHoldingAndEmit(transactionId uuid.UUID, holdingId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ReleaseFromMtsHolding(mb)(transactionId, holdingId)
	})
}

func (p *ProcessorImpl) ReleaseFromMtsHolding(mb *message.Buffer) func(transactionId uuid.UUID, holdingId uuid.UUID) error {
	return func(transactionId uuid.UUID, holdingId uuid.UUID) error {
		return mb.Put(mtsCustody.EnvCommandTopic, ReleaseFromMtsHoldingProvider(transactionId, holdingId))
	}
}

func (p *ProcessorImpl) RestoreMtsHoldingAndEmit(transactionId uuid.UUID, holdingId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RestoreMtsHolding(mb)(transactionId, holdingId)
	})
}

func (p *ProcessorImpl) RestoreMtsHolding(mb *message.Buffer) func(transactionId uuid.UUID, holdingId uuid.UUID) error {
	return func(transactionId uuid.UUID, holdingId uuid.UUID) error {
		return mb.Put(mtsCustody.EnvCommandTopic, RestoreMtsHoldingProvider(transactionId, holdingId))
	}
}

func (p *ProcessorImpl) MoveListingToHoldingAndEmit(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.MoveListingToHolding(mb)(transactionId, listingId, buyerId, worldId, resultKind, price)
	})
}

func (p *ProcessorImpl) MoveListingToHolding(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) error {
	return func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) error {
		return mb.Put(mtsCustody.EnvCommandTopic, MoveListingToHoldingProvider(transactionId, listingId, buyerId, worldId, resultKind, price))
	}
}

func (p *ProcessorImpl) RemoveMtsListingAndEmit(transactionId uuid.UUID, listingId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RemoveMtsListing(mb)(transactionId, listingId)
	})
}

func (p *ProcessorImpl) RemoveMtsListing(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID) error {
	return func(transactionId uuid.UUID, listingId uuid.UUID) error {
		return mb.Put(mtsCustody.EnvCommandTopic, RemoveMtsListingProvider(transactionId, listingId))
	}
}

func (p *ProcessorImpl) RestoreListingFromHoldingAndEmit(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RestoreListingFromHolding(mb)(transactionId, listingId, buyerId)
	})
}

func (p *ProcessorImpl) RestoreListingFromHolding(mb *message.Buffer) func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) error {
	return func(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) error {
		return mb.Put(mtsCustody.EnvCommandTopic, RestoreListingFromHoldingProvider(transactionId, listingId, buyerId))
	}
}
