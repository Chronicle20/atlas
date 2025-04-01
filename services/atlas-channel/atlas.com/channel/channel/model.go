package channel

import "github.com/google/uuid"

type Model struct {
	id        uuid.UUID
	worldId   byte
	channelId byte
	ipAddress string
	port      int
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
