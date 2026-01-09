package pet

// Model represents pet reference data fetched from atlas-pets
type Model struct {
	id          uint32
	ownerId     uint32
	cashId      int64
	flag        uint16
	purchasedBy uint32
	name        string
	level       byte
	closeness   uint16
	fullness    byte
	slot        int8
}

func (m Model) Id() uint32          { return m.id }
func (m Model) OwnerId() uint32     { return m.ownerId }
func (m Model) CashId() int64       { return m.cashId }
func (m Model) Flag() uint16        { return m.flag }
func (m Model) PurchasedBy() uint32 { return m.purchasedBy }
func (m Model) Name() string        { return m.name }
func (m Model) Level() byte         { return m.level }
func (m Model) Closeness() uint16   { return m.closeness }
func (m Model) Fullness() byte      { return m.fullness }
func (m Model) Slot() int8          { return m.slot }
