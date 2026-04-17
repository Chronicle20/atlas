package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyChannelChangeWriter = "BuddyChannelChange"

type ChannelChange struct {
	mode        byte
	characterId uint32
	channelId   int8
}

func NewBuddyChannelChange(mode byte, characterId uint32, channelId int8) ChannelChange {
	return ChannelChange{mode: mode, characterId: characterId, channelId: channelId}
}

func (m ChannelChange) Mode() byte        { return m.mode }
func (m ChannelChange) CharacterId() uint32 { return m.characterId }
func (m ChannelChange) ChannelId() int8    { return m.channelId }
func (m ChannelChange) Operation() string  { return BuddyChannelChangeWriter }

func (m ChannelChange) String() string {
	return fmt.Sprintf("channel change characterId [%d] channelId [%d]", m.characterId, m.channelId)
}

func (m ChannelChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.characterId)
		w.WriteByte(0) // inShop
		w.WriteInt32(int32(m.channelId))
		return w.Bytes()
	}
}

func (m *ChannelChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.characterId = r.ReadUint32()
		_ = r.ReadByte() // inShop
		m.channelId = int8(r.ReadInt32())
	}
}
