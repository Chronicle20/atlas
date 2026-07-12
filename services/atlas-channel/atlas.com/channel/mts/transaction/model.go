package transaction

import "time"

// Model is the channel-side view of an atlas-mts transaction-history row, read
// over REST for the My Page -> History list (ITC section 4 / sub 2). Only the
// fields the History view renders into an ITCITEM are consumed here.
type Model struct {
	id         string
	itemId     uint32
	quantity   uint32
	totalPrice uint32
	kind       string
	createdAt  time.Time
}

func (m Model) Id() string           { return m.id }
func (m Model) ItemId() uint32       { return m.itemId }
func (m Model) Quantity() uint32     { return m.quantity }
func (m Model) TotalPrice() uint32   { return m.totalPrice }
func (m Model) Kind() string         { return m.kind }
func (m Model) CreatedAt() time.Time { return m.createdAt }
