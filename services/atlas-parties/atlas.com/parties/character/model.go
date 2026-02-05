package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	tenantId uuid.UUID
	id       uint32
	name     string
	level    byte
	jobId    job.Id
	field    field.Model
	partyId  uint32
	online   bool
	gm       int
}

func (m Model) LeaveParty() Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  0,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) JoinParty(partyId uint32) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  partyId,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) ChangeMap(mapId _map.Id) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field.Clone().SetMapId(mapId).Build(),
		partyId:  m.partyId,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) ChangeChannel(channelId channel.Id) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field.Clone().SetChannelId(channelId).Build(),
		partyId:  m.partyId,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) Logout() Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  m.partyId,
		online:   false,
		gm:       m.gm,
	}
}

func (m Model) Login() Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  m.partyId,
		online:   true,
		gm:       m.gm,
	}
}

func (m Model) ChangeLevel(level byte) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  m.partyId,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) ChangeJob(jobId job.Id) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    jobId,
		field:    m.field,
		partyId:  m.partyId,
		online:   m.online,
		gm:       m.gm,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) JobId() job.Id {
	return m.jobId
}

func (m Model) Field() field.Model {
	return m.field
}

func (m Model) WorldId() world.Id {
	return m.Field().WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.Field().ChannelId()
}

func (m Model) MapId() _map.Id {
	return m.Field().MapId()
}

func (m Model) Instance() uuid.UUID {
	return m.Field().Instance()
}

func (m Model) Online() bool {
	return m.online
}

func (m Model) PartyId() uint32 {
	return m.partyId
}

func (m Model) GM() int {
	return m.gm
}

type ForeignModel struct {
	id      uint32
	worldId world.Id
	mapId   _map.Id
	name    string
	level   byte
	jobId   job.Id
	gm      int
}

func (m ForeignModel) Name() string {
	return m.name
}

func (m ForeignModel) Level() byte {
	return m.level
}

func (m ForeignModel) JobId() job.Id {
	return m.jobId
}

func (m ForeignModel) WorldId() world.Id {
	return m.worldId
}

func (m ForeignModel) MapId() _map.Id {
	return m.mapId
}

func (m ForeignModel) GM() int {
	return m.gm
}
