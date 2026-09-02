package object

// Model is a named object declared by a map's `obj` nodes. State is the
// object's declared default appearance -- what the client shows before any
// script or command changes it.
type Model struct {
	name  string
	state uint32
}

func (m Model) Name() string {
	return m.name
}

func (m Model) State() uint32 {
	return m.state
}
