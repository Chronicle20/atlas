package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type modelBuilder struct {
	id      uint32
	members []MemberModel
}

// NewBuilder returns a party model builder. The production path builds
// through Extract over the REST response; this exists for tests and any
// in-process construction.
func NewBuilder() *modelBuilder {
	return &modelBuilder{members: make([]MemberModel, 0)}
}

func (b *modelBuilder) SetId(v uint32) *modelBuilder             { b.id = v; return b }
func (b *modelBuilder) SetMembers(v []MemberModel) *modelBuilder { b.members = v; return b }

func (b *modelBuilder) Build() Model {
	return Model{
		id:      b.id,
		members: b.members,
	}
}

type memberBuilder struct {
	id     uint32
	field  field.Model
	online bool
}

func NewMemberBuilder() *memberBuilder {
	return &memberBuilder{}
}

func (b *memberBuilder) SetId(v uint32) *memberBuilder         { b.id = v; return b }
func (b *memberBuilder) SetField(v field.Model) *memberBuilder { b.field = v; return b }
func (b *memberBuilder) SetOnline(v bool) *memberBuilder       { b.online = v; return b }

func (b *memberBuilder) Build() MemberModel {
	return MemberModel{
		id:     b.id,
		field:  b.field,
		online: b.online,
	}
}
