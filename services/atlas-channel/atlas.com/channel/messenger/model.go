package messenger

import (
	"errors"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	id      uint32
	members []MemberModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

func (m Model) FindMember(id uint32) (MemberModel, error) {
	for _, mm := range m.Members() {
		if mm.Id() == id {
			return mm, nil
		}
	}
	return MemberModel{}, errors.New("not found")
}

type MemberModel struct {
	id        uint32
	name      string
	worldId   world.Id
	channelId channel.Id
	online    bool
	slot      byte
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Name() string {
	return m.name
}

func (m MemberModel) Online() bool {
	return m.online
}

func (m MemberModel) ChannelId() channel.Id {
	return m.channelId
}

func (m MemberModel) Slot() byte {
	return m.slot
}
