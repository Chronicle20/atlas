package cash

// Model is a cash item template's pet-revival attributes, as resolved from
// atlas-data.
type Model struct {
	id   uint32
	life uint32
}

func (m Model) Id() uint32 { return m.id }

// Life is info/life in DAYS — the lifespan a Water of Life grants a revived
// pet. Zero means the WZ node is absent, which the revive treats as a data
// error (reject, consume nothing).
func (m Model) Life() uint32 { return m.life }
