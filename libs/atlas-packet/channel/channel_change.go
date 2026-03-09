package channel

import (
	"context"
	"fmt"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ChannelChangeHandle = "ChannelChangeHandle"

// ChannelChange - CField::SendTransferChannelRequest
type ChannelChange struct {
	channelId  channel2.Id
	updateTime uint32
}

func (m ChannelChange) ChannelId() channel2.Id {
	return m.channelId
}

func (m ChannelChange) UpdateTime() uint32 {
	return m.updateTime
}

func (m ChannelChange) Operation() string {
	return ChannelChangeHandle
}

func (m ChannelChange) String() string {
	return fmt.Sprintf("channelId [%d], updateTime [%d]", m.channelId, m.updateTime)
}

func (m ChannelChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.channelId))
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ChannelChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.channelId = channel2.Id(r.ReadByte())
		m.updateTime = r.ReadUint32()
	}
}
