package channel

import (
	"atlas-channel/tenant"
)

const (
	EnvTopicTokenStatus = "TOPIC_CHANNEL_SERVICE"

	EventStatusStarted  = "STARTED"
	EventStatusShutdown = "SHUTDOWN"
)

type channelServerEvent struct {
	Tenant    tenant.Model `json:"tenant"`
	Status    string       `json:"status"`
	WorldId   byte         `json:"worldId"`
	ChannelId byte         `json:"channelId"`
	IpAddress string       `json:"ipAddress"`
	Port      int          `json:"port"`
}
