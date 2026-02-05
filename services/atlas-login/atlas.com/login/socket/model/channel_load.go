package model

import "github.com/Chronicle20/atlas-constants/channel"

type Load struct {
	channelId channel.Id
	capacity  uint32
}

func NewChannelLoad(channelId channel.Id, capacity uint32) Load {
	return Load{channelId, capacity}
}

func (cl Load) ChannelId() channel.Id {
	return cl.channelId
}

func (cl Load) Capacity() uint32 {
	return cl.capacity
}
