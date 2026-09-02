package object

type Builder struct{ m Model }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetName(v string) *Builder  { b.m.name = v; return b }
func (b *Builder) SetState(v uint32) *Builder { b.m.state = v; return b }
func (b *Builder) Build() Model               { return b.m }
