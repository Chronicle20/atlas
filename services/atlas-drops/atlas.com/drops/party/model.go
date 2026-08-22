package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Model is the subset of a party atlas-drops needs: the roster, with each
// member's location and online flag. Name, level, job, and leadership are
// deliberately absent — the meso split does not read them.
type Model struct {
	id      uint32
	members []MemberModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

type MemberModel struct {
	id     uint32
	field  field.Model
	online bool
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Field() field.Model {
	return m.field
}

func (m MemberModel) Online() bool {
	return m.online
}
