package channel

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic          topic.Token = "COMMAND_TOPIC_CHANNEL_STATUS"
	CommandTypeStatusRequest             = "STATUS_REQUEST"
)

type StatusCommand struct {
	Type string `json:"type"`
}

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_CHANNEL_STATUS"
)

type StatusEvent struct {
	Type            channel.StatusType `json:"type"`
	WorldId         world.Id           `json:"worldId"`
	ChannelId       channel.Id         `json:"channelId"`
	IpAddress       string             `json:"ipAddress"`
	Port            int                `json:"port"`
	CurrentCapacity uint32             `json:"currentCapacity"`
	MaxCapacity     uint32             `json:"maxCapacity"`
}
