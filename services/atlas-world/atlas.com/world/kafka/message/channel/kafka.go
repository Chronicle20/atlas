package channel

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHANNEL_STATUS"
	CommandTypeStatusRequest = "STATUS_REQUEST"
)

type StatusCommand struct {
	Type string `json:"type"`
}

const (
	EnvEventTopicStatus = "EVENT_TOPIC_CHANNEL_STATUS"
	StatusTypeStarted   = "STARTED"
	StatusTypeShutdown  = "SHUTDOWN"
)

type StatusEvent struct {
	Type            string `json:"type"`
	WorldId         byte   `json:"worldId"`
	ChannelId       byte   `json:"channelId"`
	IpAddress       string `json:"ipAddress"`
	Port            int    `json:"port"`
	CurrentCapacity uint32 `json:"currentCapacity"`
	MaxCapacity     uint32 `json:"maxCapacity"`
}
