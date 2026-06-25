package transaction

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Kind enumerates which side of a settled listing a transaction-history row
// represents: a purchase row (the buyer's view) or a sale row (the seller's).
const (
	KindPurchase = "purchase"
	KindSale     = "sale"
)

// Model is the immutable MTS transaction-history record. A settle writes two:
// the buyer's purchase row and the seller's sale row. Construct it via the
// Builder.
type Model struct {
	id             uuid.UUID
	tenantId       uuid.UUID
	worldId        world.Id
	characterId    uint32
	counterpartyId uint32
	itemId         uint32
	quantity       uint32
	totalPrice     uint32
	kind           string
	createdAt      time.Time
}

func (m Model) Id() uuid.UUID          { return m.id }
func (m Model) TenantId() uuid.UUID    { return m.tenantId }
func (m Model) WorldId() world.Id      { return m.worldId }
func (m Model) CharacterId() uint32    { return m.characterId }
func (m Model) CounterpartyId() uint32 { return m.counterpartyId }
func (m Model) ItemId() uint32         { return m.itemId }
func (m Model) Quantity() uint32       { return m.quantity }
func (m Model) TotalPrice() uint32     { return m.totalPrice }
func (m Model) Kind() string           { return m.kind }
func (m Model) CreatedAt() time.Time   { return m.createdAt }
