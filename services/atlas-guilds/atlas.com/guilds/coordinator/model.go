package coordinator

import "time"

type Model struct {
	worldId   byte
	channelId byte
	leaderId  uint32
	name      string
	requests  []uint32
	responses map[uint32]bool
	age       time.Time
}

func (m Model) Agree(characterId uint32) Model {
	m.responses[characterId] = true
	return Model{
		worldId:   m.worldId,
		channelId: m.channelId,
		leaderId:  m.leaderId,
		name:      m.name,
		requests:  m.requests,
		responses: m.responses,
		age:       m.age,
	}
}

func (m Model) Responses() map[uint32]bool {
	return m.responses
}

func (m Model) Requests() []uint32 {
	return m.requests
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) WorldId() byte {
	return m.worldId
}
