package character

import (
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/google/uuid"
)

type Model struct {
	tenantId  uuid.UUID
	id        uint32
	name      string
	level     byte
	jobId     job.Id
	worldId   byte
	channelId byte
	mapId     uint32
	partyId   uint32
	online    bool
	gm        int
}

func (m Model) LeaveParty() Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   0,
		online:    m.online,
		gm:        m.gm,
	}
}

func (m Model) JoinParty(partyId uint32) Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   partyId,
		online:    m.online,
		gm:        m.gm,
	}
}

func (m Model) ChangeMap(mapId uint32) Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     mapId,
		partyId:   m.partyId,
		online:    m.online,
		gm:        m.gm,
	}
}

func (m Model) ChangeChannel(channelId byte) Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: channelId,
		mapId:     m.mapId,
		partyId:   m.partyId,
		online:    m.online,
		gm:        m.gm,
	}
}

func (m Model) Logout() Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   m.partyId,
		online:    false,
		gm:        m.gm,
	}
}

func (m Model) Login() Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   m.partyId,
		online:    true,
		gm:        m.gm,
	}
}

func (m Model) ChangeLevel(level byte) Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     level,
		jobId:     m.jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   m.partyId,
		online:    m.online,
		gm:        m.gm,
	}
}

func (m Model) ChangeJob(jobId job.Id) Model {
	return Model{
		tenantId:  m.tenantId,
		id:        m.id,
		name:      m.name,
		level:     m.level,
		jobId:     jobId,
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		partyId:   m.partyId,
		online:    m.online,
		gm:        m.gm,
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

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) MapId() uint32 {
	return m.mapId
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
	worldId byte
	mapId   uint32
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

func (m ForeignModel) WorldId() byte {
	return m.worldId
}

func (m ForeignModel) MapId() uint32 {
	return m.mapId
}

func (m ForeignModel) GM() int {
	return m.gm
}
