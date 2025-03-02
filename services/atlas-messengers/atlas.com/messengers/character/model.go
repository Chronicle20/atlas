package character

import (
	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uint32
	name        string
	worldId     byte
	channelId   byte
	messengerId uint32
	online      bool
}

func (m Model) LeaveMessenger() Model {
	return Model{
		tenantId:    m.tenantId,
		id:          m.id,
		name:        m.name,
		worldId:     m.worldId,
		channelId:   m.channelId,
		messengerId: 0,
		online:      m.online,
	}
}

func (m Model) JoinMessenger(messengerId uint32) Model {
	return Model{
		tenantId:    m.tenantId,
		id:          m.id,
		name:        m.name,
		worldId:     m.worldId,
		channelId:   m.channelId,
		messengerId: messengerId,
		online:      m.online,
	}
}

func (m Model) ChangeChannel(channelId byte) Model {
	return Model{
		tenantId:    m.tenantId,
		id:          m.id,
		name:        m.name,
		worldId:     m.worldId,
		channelId:   channelId,
		messengerId: m.messengerId,
		online:      m.online,
	}
}

func (m Model) Logout() Model {
	return Model{
		tenantId:    m.tenantId,
		id:          m.id,
		name:        m.name,
		worldId:     m.worldId,
		channelId:   m.channelId,
		messengerId: m.messengerId,
		online:      false,
	}
}

func (m Model) Login() Model {
	return Model{
		tenantId:    m.tenantId,
		id:          m.id,
		name:        m.name,
		worldId:     m.worldId,
		channelId:   m.channelId,
		messengerId: m.messengerId,
		online:      true,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) Online() bool {
	return m.online
}

func (m Model) MessengerId() uint32 {
	return m.messengerId
}

type ForeignModel struct {
	id      uint32
	worldId byte
	mapId   uint32
	name    string
	level   byte
	jobId   uint16
	gm      int
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

func (m ForeignModel) GM() int {
	return m.gm
}
