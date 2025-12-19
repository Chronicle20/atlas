package model

type ChannelSelect struct {
	IPAddress   string `json:"ipAddress"`
	Port        uint16 `json:"port"`
	CharacterId uint32 `json:"characterId"`
}
