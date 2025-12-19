package model

type Load struct {
	channelId byte
	capacity  uint32
}

func NewChannelLoad(channelId byte, capacity uint32) Load {
	return Load{channelId, capacity}
}

func (cl Load) ChannelId() byte {
	return cl.channelId
}

func (cl Load) Capacity() uint32 {
	return cl.capacity
}
