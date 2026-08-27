package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type Model struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Members() []MemberModel {
	return m.members
}

type MemberModel struct {
	id     uint32
	name   string
	level  byte
	jobId  job.Id
	field  field.Model
	online bool
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Name() string {
	return m.name
}

func (m MemberModel) Level() byte {
	return m.level
}

func (m MemberModel) JobId() job.Id {
	return m.jobId
}

func (m MemberModel) Field() field.Model {
	return m.field
}

func (m MemberModel) Online() bool {
	return m.online
}
