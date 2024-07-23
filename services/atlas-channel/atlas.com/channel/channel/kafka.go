package channel

import "atlas-channel/tenant"

const (
	EnvCommandTopicChannelStatus = "COMMAND_TOPIC_CHANNEL_STATUS"
	CommandChannelStatusType     = "STATUS_REQUEST"
)

type channelStatusCommand struct {
	Tenant tenant.Model `json:"tenant"`
	Type   string       `json:"type"`
}
