package channel

type Model struct {
	worldId   byte
	channelId byte
	capacity  int
	ipAddress string
	port      uint16
}

func (c Model) WorldId() byte {
	return c.worldId
}

func (c Model) ChannelId() byte {
	return c.channelId
}

func (c Model) Capacity() int {
	return c.capacity
}

func (c Model) IpAddress() string {
	return c.ipAddress
}

func (c Model) Port() uint16 {
	return c.port
}

type channelBuilder struct {
	worldId   byte
	channelId byte
	capacity  int
	ipAddress string
	port      uint16
}

func NewChannelBuilder() *channelBuilder {
	return &channelBuilder{}
}

func (c *channelBuilder) SetWorldId(worldId byte) *channelBuilder {
	c.worldId = worldId
	return c
}

func (c *channelBuilder) SetChannelId(channelId byte) *channelBuilder {
	c.channelId = channelId
	return c
}

func (c *channelBuilder) SetCapacity(capacity int) *channelBuilder {
	c.capacity = capacity
	return c
}

func (c *channelBuilder) SetIpAddress(ipAddress string) *channelBuilder {
	c.ipAddress = ipAddress
	return c
}

func (c *channelBuilder) SetPort(port uint16) *channelBuilder {
	c.port = port
	return c
}

func (c *channelBuilder) Build() Model {
	return Model{
		worldId:   c.worldId,
		channelId: c.channelId,
		capacity:  c.capacity,
		ipAddress: c.ipAddress,
		port:      c.port,
	}
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
