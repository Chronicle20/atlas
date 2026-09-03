package npc

// Builder constructs a Model. Tests and callers use it instead of a
// test-only constructor to keep the immutable-model pattern consistent.
type Builder struct {
	uniqueId uint32
	npcId    uint32
	x        int16
	y        int16
	fh       int16
}

func NewBuilder(uniqueId uint32, npcId uint32) *Builder {
	return &Builder{
		uniqueId: uniqueId,
		npcId:    npcId,
	}
}

func (b *Builder) SetPosition(x int16, y int16, fh int16) *Builder {
	b.x = x
	b.y = y
	b.fh = fh
	return b
}

func (b *Builder) Build() Model {
	return Model{
		uniqueId: b.uniqueId,
		npcId:    b.npcId,
		x:        b.x,
		y:        b.y,
		fh:       b.fh,
	}
}
