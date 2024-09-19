package channel

const (
	EnvCommandTopicChannelStatus = "COMMAND_TOPIC_CHANNEL_STATUS"
	CommandChannelStatusType     = "STATUS_REQUEST"
)

type channelStatusCommand struct {
	Type string `json:"type"`
}
