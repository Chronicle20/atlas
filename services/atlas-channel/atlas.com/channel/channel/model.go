package channel

type Model struct {
	id        byte
	ipAddress string
	port      int
}

func (c Model) IpAddress() string {
	return c.ipAddress
}

func (c Model) Port() int {
	return c.port
}
