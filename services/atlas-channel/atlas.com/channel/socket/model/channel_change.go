package model

type ChannelChange struct {
	IPAddress string `json:"ipAddress"`
	Port      uint16 `json:"port"`
}
