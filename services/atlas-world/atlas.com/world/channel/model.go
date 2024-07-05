package channel

type Model struct {
	uniqueId  uint32
	worldId   byte
	channelId byte
	ipAddress string
	port      int
}

func (c Model) UniqueId() uint32 {
	return c.uniqueId
}

func (c Model) WorldId() byte {
	return c.worldId
}

func (c Model) ChannelId() byte {
	return c.channelId
}

func (c Model) IpAddress() string {
	return c.ipAddress
}

func (c Model) Port() int {
	return c.port
}

func NewModel(uniqueId uint32, worldId byte, channelId byte, ipAddress string, port int) Model {
	return Model{
		uniqueId:  uniqueId,
		worldId:   worldId,
		channelId: channelId,
		ipAddress: ipAddress,
		port:      port,
	}
}
