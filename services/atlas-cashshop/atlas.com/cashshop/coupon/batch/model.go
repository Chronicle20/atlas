package batch

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Model is one bulk generation.
//
// redeemedCount is NOT a stored column: it is counted from the redemption rows
// of the batch's coupons when the batch is read, so it can never drift from
// the audit trail the way a denormalized counter would.
type Model struct {
	id             uuid.UUID
	description    string
	requestedCount uint32
	generatedCount uint32
	redeemedCount  uint32
	createdAt      time.Time
}

func (m Model) Id() uuid.UUID          { return m.id }
func (m Model) Description() string    { return m.description }
func (m Model) RequestedCount() uint32 { return m.requestedCount }
func (m Model) GeneratedCount() uint32 { return m.generatedCount }
func (m Model) RedeemedCount() uint32  { return m.redeemedCount }
func (m Model) CreatedAt() time.Time   { return m.createdAt }

var ErrInvalidBatch = errors.New("invalid batch")
