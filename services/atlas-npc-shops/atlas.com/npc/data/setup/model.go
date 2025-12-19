package setup

type Model struct {
	id         uint32
	price      uint32
	slotMax    uint32
	recoveryHP uint32
	tradeBlock bool
	notSale    bool
	reqLevel   uint32
	distanceX  uint32
	distanceY  uint32
	maxDiff    uint32
	direction  uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Price() uint32 {
	return m.price
}

func (m Model) SlotMax() uint32 {
	return m.slotMax
}

func (m Model) RecoveryHP() uint32 {
	return m.recoveryHP
}

func (m Model) TradeBlock() bool {
	return m.tradeBlock
}

func (m Model) NotSale() bool {
	return m.notSale
}

func (m Model) ReqLevel() uint32 {
	return m.reqLevel
}

func (m Model) DistanceX() uint32 {
	return m.distanceX
}

func (m Model) DistanceY() uint32 {
	return m.distanceY
}

func (m Model) MaxDiff() uint32 {
	return m.maxDiff
}

func (m Model) Direction() uint32 {
	return m.direction
}