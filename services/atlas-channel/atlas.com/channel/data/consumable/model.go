package consumable

// Model is a consumable's recovery spec, as resolved from atlas-data.
type Model struct {
	id          uint32
	spec        map[SpecType]int32
	npc         uint32
	script      string
	runOnPickup bool
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}

// Npc is the NPC template a scripted item's dialogue renders with (the
// 243xxxx family) or the NPC a remote-NPC item summons (the 239xxxx family).
func (m Model) Npc() uint32 {
	return m.npc
}

// Script is the WZ spec/script value, recorded for authoring traceability
// only; conversations are keyed by item id, never by script name.
func (m Model) Script() string {
	return m.script
}

func (m Model) RunOnPickup() bool {
	return m.runOnPickup
}
