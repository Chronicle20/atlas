package npc

type Model struct {
	id       uint32
	template uint32
	x        int16
	cy       int16
	f        uint32
	fh       uint16
	rx0      int16
	rx1      int16
}

func (n Model) Id() uint32 {
	return n.id
}

func (n Model) X() int16 {
	return n.x
}

func (n Model) CY() int16 {
	return n.cy
}

func (n Model) F() uint32 {
	return n.f
}

func (n Model) Fh() uint16 {
	return n.fh
}

func (n Model) RX0() int16 {
	return n.rx0
}

func (n Model) RX1() int16 {
	return n.rx1
}

func (n Model) Template() uint32 {
	return n.template
}
