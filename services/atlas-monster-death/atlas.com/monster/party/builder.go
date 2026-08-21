package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Builder constructs a party Model for use in tests and other in-process
// call sites. Production code populates a Model via Extract over the REST
// response.
type Builder struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func NewBuilder(id uint32) *Builder {
	return &Builder{
		id: id,
	}
}

func (b *Builder) SetLeaderId(leaderId uint32) *Builder {
	b.leaderId = leaderId
	return b
}

func (b *Builder) AddMember(m MemberModel) *Builder {
	b.members = append(b.members, m)
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:       b.id,
		leaderId: b.leaderId,
		members:  b.members,
	}
}

// MemberBuilder constructs a party MemberModel for use in tests and other
// in-process call sites.
type MemberBuilder struct {
	id     uint32
	name   string
	level  byte
	jobId  job.Id
	field  field.Model
	online bool
}

func NewMemberBuilder(id uint32) *MemberBuilder {
	return &MemberBuilder{
		id: id,
	}
}

func (b *MemberBuilder) SetName(name string) *MemberBuilder {
	b.name = name
	return b
}

func (b *MemberBuilder) SetLevel(level byte) *MemberBuilder {
	b.level = level
	return b
}

func (b *MemberBuilder) SetJobId(jobId job.Id) *MemberBuilder {
	b.jobId = jobId
	return b
}

func (b *MemberBuilder) SetField(f field.Model) *MemberBuilder {
	b.field = f
	return b
}

func (b *MemberBuilder) SetOnline(online bool) *MemberBuilder {
	b.online = online
	return b
}

func (b *MemberBuilder) Build() MemberModel {
	return MemberModel{
		id:     b.id,
		name:   b.name,
		level:  b.level,
		jobId:  b.jobId,
		field:  b.field,
		online: b.online,
	}
}
