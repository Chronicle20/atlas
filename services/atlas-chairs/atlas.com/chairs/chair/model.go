package chair

type Model struct {
	worldId   byte
	channelId byte
	mapId     uint32
	id        uint32
	chairType string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) Type() string {
	return m.chairType
}
