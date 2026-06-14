package party

// Model is a minimal party representation for atlas-doors.
// Members() preserves the join-order slice returned by atlas-parties
// (leader-seeded at index 0, then join order). The doors slot assignment
// relies on this stable index — do NOT re-sort.
type Model struct {
	id       uint32
	leaderId uint32
	members  []uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

// Members returns the ordered member character-id slice (join order,
// leader at index 0).
func (m Model) Members() []uint32 {
	return m.members
}
