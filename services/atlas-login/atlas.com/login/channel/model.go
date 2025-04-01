package channel

import "github.com/google/uuid"

type Model struct {
	id        uuid.UUID
	worldId   byte
	channelId byte
	ipAddress string
	port      int
	capacity  int
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) IpAddress() string {
	return m.ipAddress
}

func (m Model) Port() int {
	return m.port
}

func (m Model) Capacity() int {
	return m.capacity
}

type Load struct {
	channelId byte
	capacity  int
}

func NewChannelLoad(channelId byte, capacity int) Load {
	return Load{channelId, capacity}
}

func (cl Load) ChannelId() byte {
	return cl.channelId
}

func (cl Load) Capacity() int {
	return cl.capacity
}
