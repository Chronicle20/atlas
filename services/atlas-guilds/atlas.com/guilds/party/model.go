package party

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Id() uint32 {
	return m.id
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

func (m MemberModel) JobId() job.Id {
	return m.jobId
}

func (m MemberModel) Level() byte {
	return m.level
}

func (m MemberModel) Online() bool {
	return m.online
}

func (m MemberModel) Field() field.Model {
	return m.field
}

func (m MemberModel) WorldId() world.Id {
	return m.Field().WorldId()
}

func (m MemberModel) ChannelId() channel.Id {
	return m.Field().ChannelId()
}

func (m MemberModel) MapId() _map.Id {
	return m.Field().MapId()
}
