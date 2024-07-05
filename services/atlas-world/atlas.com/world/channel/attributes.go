package channel

type InputDataContainer struct {
	Data DataBody `json:"data"`
}

type DataListContainer struct {
	Data []DataBody `json:"data"`
}

type DataContainer struct {
	Data DataBody `json:"data"`
}

type DataBody struct {
	Id         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	Capacity  int    `json:"capacity"`
	IpAddress string `json:"ipAddress"`
	Port      int    `json:"port"`
}
