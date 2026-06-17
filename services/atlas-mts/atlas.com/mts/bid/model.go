package bid

import (
	"time"

	"github.com/google/uuid"
)

// State enumerates the lifecycle state of a bid's escrowed funds.
type State string

const (
	StateHeld     State = "held"
	StateReleased State = "released"
	StateWon      State = "won"
)

// Model is the immutable record of a bid placed on an auction listing. The
// escrowTxnId references the held NX in the wallet ledger. Construct it via the
// Builder.
type Model struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	listingId   uuid.UUID
	bidderId    uint32
	amount      uint32
	escrowTxnId uuid.UUID
	state       State
	createdAt   time.Time
}

func (m Model) Id() uuid.UUID          { return m.id }
func (m Model) TenantId() uuid.UUID    { return m.tenantId }
func (m Model) ListingId() uuid.UUID   { return m.listingId }
func (m Model) BidderId() uint32       { return m.bidderId }
func (m Model) Amount() uint32         { return m.amount }
func (m Model) EscrowTxnId() uuid.UUID { return m.escrowTxnId }
func (m Model) State() State           { return m.state }
func (m Model) CreatedAt() time.Time   { return m.createdAt }
