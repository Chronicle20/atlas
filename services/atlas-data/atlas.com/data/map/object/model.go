package object

// Model is the immutable domain representation of one named WZ object
// declared on a map. Private fields + getters, per the project's
// immutable-model convention; construction goes exclusively through Builder.
type Model struct {
	kind         string
	name         string
	objectSource string
	l0           string
	l1           string
	l2           string
	x            int16
	y            int16
	z            int32
	layer        uint32
}

// Id is the composite "{kind}:{name}" key, deliberately the same composite
// key task-278's environment-object resource uses, so the UI merges the two
// collections by id rather than by heuristic.
func (m Model) Id() string {
	return m.kind + ":" + m.name
}

func (m Model) Kind() string         { return m.kind }
func (m Model) Name() string         { return m.name }
func (m Model) ObjectSource() string { return m.objectSource }
func (m Model) L0() string           { return m.l0 }
func (m Model) L1() string           { return m.l1 }
func (m Model) L2() string           { return m.l2 }
func (m Model) X() int16             { return m.x }
func (m Model) Y() int16             { return m.y }
func (m Model) Z() int32             { return m.z }
func (m Model) Layer() uint32        { return m.layer }

// Builder accumulates the fields of a Model.
type Builder struct {
	kind         string
	name         string
	objectSource string
	l0           string
	l1           string
	l2           string
	x            int16
	y            int16
	z            int32
	layer        uint32
}

// NewBuilder constructs an empty Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetKind(kind string) *Builder {
	b.kind = kind
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetObjectSource(objectSource string) *Builder {
	b.objectSource = objectSource
	return b
}

func (b *Builder) SetL0(l0 string) *Builder {
	b.l0 = l0
	return b
}

func (b *Builder) SetL1(l1 string) *Builder {
	b.l1 = l1
	return b
}

func (b *Builder) SetL2(l2 string) *Builder {
	b.l2 = l2
	return b
}

func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

func (b *Builder) SetZ(z int32) *Builder {
	b.z = z
	return b
}

func (b *Builder) SetLayer(layer uint32) *Builder {
	b.layer = layer
	return b
}

// Build materializes the immutable domain Model from the builder's
// accumulated state.
func (b *Builder) Build() Model {
	return Model{
		kind:         b.kind,
		name:         b.name,
		objectSource: b.objectSource,
		l0:           b.l0,
		l1:           b.l1,
		l2:           b.l2,
		x:            b.x,
		y:            b.y,
		z:            b.z,
		layer:        b.layer,
	}
}
