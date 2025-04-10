package stackable

type Model struct {
	id           uint32
	quantity     uint32
	owner        string
	flag         uint16
	rechargeable uint64
}

type ModelBuilder struct {
	id           uint32
	quantity     uint32
	owner        string
	flag         uint16
	rechargeable uint64
}

func (mb *ModelBuilder) SetID(id uint32) *ModelBuilder {
	mb.id = id
	return mb
}

func (mb *ModelBuilder) SetQuantity(quantity uint32) *ModelBuilder {
	mb.quantity = quantity
	return mb
}

func (mb *ModelBuilder) SetOwner(owner string) *ModelBuilder {
	mb.owner = owner
	return mb
}

func (mb *ModelBuilder) SetFlag(flag uint16) *ModelBuilder {
	mb.flag = flag
	return mb
}

func (mb *ModelBuilder) SetRechargeable(rechargeable uint64) *ModelBuilder {
	mb.rechargeable = rechargeable
	return mb
}

func (mb *ModelBuilder) Build() Model {
	return Model{
		id:           mb.id,
		quantity:     mb.quantity,
		owner:        mb.owner,
		flag:         mb.flag,
		rechargeable: mb.rechargeable,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Owner() string {
	return m.owner
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) Rechargeable() uint64 {
	return m.rechargeable
}
