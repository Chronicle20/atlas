package model

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

type ChannelLoad struct {
	channelId channel.Id
	capacity  uint32
}

func NewChannelLoad(channelId channel.Id, capacity uint32) ChannelLoad {
	return ChannelLoad{channelId: channelId, capacity: capacity}
}

func (m ChannelLoad) ChannelId() channel.Id { return m.channelId }
func (m ChannelLoad) Capacity() uint32      { return m.capacity }

func (m ChannelLoad) Write(w *response.Writer) {
	w.WriteShort(uint16(m.channelId))
	w.WriteInt(m.capacity)
}

func (m *ChannelLoad) Read(r *request.Reader) {
	m.channelId = channel.Id(r.ReadUint16())
	m.capacity = r.ReadUint32()
}
