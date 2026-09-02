package channel

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopicChannelStatus topic.Token = "COMMAND_TOPIC_CHANNEL_STATUS"
)

const (
	CommandChannelStatusType = "STATUS_REQUEST"
)

type ChannelStatusCommand struct {
	Type string `json:"type"`
}
