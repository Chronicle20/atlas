package character

import "github.com/google/uuid"

type Model struct {
	tenantId  uuid.UUID
	id        uint32
	name      string
	level     byte
	jobId     uint16
	worldId   byte
	channelId byte
	mapId     uint32
	partyId   uint32
	online    bool
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

func (m Model) JobId() uint16 {
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

type ForeignModel struct {
	id      uint32
	worldId byte
	mapId   uint32
	name    string
	level   byte
	jobId   uint16
}

func (m ForeignModel) Name() string {
	return m.name
}

func (m ForeignModel) Level() byte {
	return m.level
}

func (m ForeignModel) JobId() uint16 {
	return m.jobId
}

func (m ForeignModel) WorldId() byte {
	return m.worldId
}

func (m ForeignModel) MapId() uint32 {
	return m.mapId
}
