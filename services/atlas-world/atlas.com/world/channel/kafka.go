package channel

const (
	EnvEventTopicChannelStatus = "EVENT_TOPIC_CHANNEL_STATUS"

	EventChannelStatusType         = "STARTED"
	EventChannelStatusTypeShutdown = "SHUTDOWN"

	EnvCommandTopicChannelStatus = "COMMAND_TOPIC_CHANNEL_STATUS"
	CommandChannelStatusType     = "STATUS_REQUEST"
)

type channelStatusEvent struct {
	Type      string `json:"type"`
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	IpAddress string `json:"ipAddress"`
	Port      int    `json:"port"`
}

type channelStatusCommand struct {
	Type string `json:"type"`
}
