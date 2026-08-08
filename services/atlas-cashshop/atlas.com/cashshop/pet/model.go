package pet

type Model struct {
	id         uint32
	cashId     uint64
	templateId uint32
	name       string
	ownerId    uint32
}

// CashId is the cash serial the pet shares with the cash-shop asset that wraps
// it. The client uses one value (GW_ItemSlotBase::liCashItemSN) to both bind a
// spawned pet to its inventory slot and to clear the locker entry on withdraw,
// so the pet row and the asset row must agree on it from the moment of purchase.
func (m Model) CashId() uint64 {
	return m.cashId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}
