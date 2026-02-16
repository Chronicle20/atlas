package party

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func (m Model) Id() uint32            { return m.id }
func (m Model) LeaderId() uint32      { return m.leaderId }
func (m Model) Members() []MemberModel { return m.members }

type MemberModel struct {
	id        uint32
	worldId   world.Id
	channelId channel.Id
}

func (m MemberModel) Id() uint32            { return m.id }
func (m MemberModel) WorldId() world.Id     { return m.worldId }
func (m MemberModel) ChannelId() channel.Id { return m.channelId }
