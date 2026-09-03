package environment

// Model is one tracked environment object entry as atlas-channel sees it,
// mirroring atlas-maps' map/environment.ObjectEntry (Kind/Name/State).
type Model struct {
	kind  string
	name  string
	state uint32
}

func (m Model) Kind() string {
	return m.kind
}

func (m Model) Name() string {
	return m.name
}

func (m Model) State() uint32 {
	return m.state
}
