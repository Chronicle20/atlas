package party

import "github.com/Chronicle20/atlas/libs/atlas-constants/character"

// Model is a minimal party representation for atlas-doors.
// Members() preserves the join-order slice returned by atlas-parties
// (leader-seeded at index 0, then join order). The doors slot assignment
// relies on this stable index — do NOT re-sort.
type Model struct {
	id       uint32
	leaderId character.Id
	members  []character.Id
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) LeaderId() character.Id {
	return m.leaderId
}

// Members returns the ordered member character-id slice (join order,
// leader at index 0).
func (m Model) Members() []character.Id {
	return m.members
}
