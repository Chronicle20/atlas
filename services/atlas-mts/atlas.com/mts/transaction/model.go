package transaction

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Kind enumerates what a transaction-history row represents. The channel maps
// each to the client's CITCItem nProcessStatus (IDA-verified v83
// CITCWnd_List::GetContractHistoryCode): purchase->1, sale->0, bid_lost->2,
// cancelled->3.
const (
	KindPurchase  = "purchase"
	KindSale      = "sale"
	KindBidLost   = "bid_lost"  // an outbid bidder's lost-bid row (nProcessStatus 2)
	KindCancelled = "cancelled" // a seller's cancelled-listing row (nProcessStatus 3)
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
