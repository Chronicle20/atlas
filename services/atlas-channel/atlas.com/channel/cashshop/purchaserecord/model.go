package purchaserecord

type Model struct {
	serialNumber uint32
	purchased    bool
	count        uint32
}

func (m Model) SerialNumber() uint32 {
	return m.serialNumber
}

func (m Model) Purchased() bool {
	return m.purchased
}

func (m Model) Count() uint32 {
	return m.count
}
