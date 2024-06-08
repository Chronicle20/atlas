package session

type dataListContainer struct {
	Data []dataBody `json:"data"`
}

type dataBody struct {
	Id         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	AccountId uint32 `json:"accountId"`
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
}
