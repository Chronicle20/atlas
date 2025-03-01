package party

import (
	"github.com/Chronicle20/atlas-constants/channel"
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

func (m Model) Leader() MemberModel {
	for _, mm := range m.Members() {
		if mm.Id() == m.LeaderId() {
			return mm
		}
	}
	return MemberModel{}
}

func (m Model) LeaderName() string {
	for _, mm := range m.Members() {
		if mm.Id() == m.LeaderId() {
			return mm.Name()
		}
	}
	return ""
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

type MemberModel struct {
	id        uint32
	name      string
	level     byte
	jobId     uint16
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	online    bool
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Name() string {
	return m.name
}

func (m MemberModel) JobId() uint16 {
	return m.jobId
}

func (m MemberModel) Level() byte {
	return m.level
}

func (m MemberModel) Online() bool {
	return m.online
}

func (m MemberModel) ChannelId() channel.Id {
	return m.channelId
}

func (m MemberModel) MapId() _map.Id {
	return m.mapId
}
