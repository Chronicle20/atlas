package cash

// Model is a cash item template's expiration-extender attributes, as resolved
// from atlas-data.
type Model struct {
	id      uint32
	addTime uint32
	maxDays uint32
	life    uint32
}

func (m Model) Id() uint32 { return m.id }

// AddTime is the expiration grant in SECONDS.
func (m Model) AddTime() uint32 { return m.addTime }

// MaxDays is the ceiling in DAYS, anchored to now.
func (m Model) MaxDays() uint32 { return m.maxDays }

// Life is info/life in DAYS — the lifespan a pet-revival item (Water of Life,
// classification 518) grants. 0518.img carries life but NO maxDays, which is
// exactly why EXTEND_EXPIRATION's maxDays ceiling cannot serve the revive flow.
func (m Model) Life() uint32 { return m.life }
