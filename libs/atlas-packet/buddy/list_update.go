package buddy

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyListUpdateWriter = "BuddyListUpdate"

type BuddyEntry struct {
	CharacterId uint32
	Name        string
	ChannelId   channel.Id
	Group       string
	InShop      bool
}

type ListUpdate struct {
	mode    byte
	buddies []BuddyEntry
}

func NewBuddyListUpdate(mode byte, buddies []BuddyEntry) ListUpdate {
	return ListUpdate{mode: mode, buddies: buddies}
}

func (m ListUpdate) Mode() byte          { return m.mode }
func (m ListUpdate) Buddies() []BuddyEntry { return m.buddies }
func (m ListUpdate) Operation() string    { return BuddyListUpdateWriter }

func (m ListUpdate) String() string {
	return fmt.Sprintf("list update with [%d] buddies", len(m.buddies))
}

func (m ListUpdate) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.buddies)))
		for _, b := range m.buddies {
			bm := model.Buddy{
				FriendId:    b.CharacterId,
				FriendName:  b.Name,
				Flag:        0,
				ChannelId:   b.ChannelId,
				FriendGroup: b.Group,
			}
			w.WriteByteArray(bm.Encode(l, ctx)(options))
		}
		for _, b := range m.buddies {
			if b.InShop {
				w.WriteInt(uint32(1))
			} else {
				w.WriteInt(uint32(0))
			}
		}
		return w.Bytes()
	}
}

func (m *ListUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.buddies = make([]BuddyEntry, count)
		for i := range m.buddies {
			m.buddies[i].CharacterId = r.ReadUint32()
			m.buddies[i].Name = model.ReadPaddedString(r, 13)
			_ = r.ReadByte() // flag
			m.buddies[i].ChannelId = channel.Id(r.ReadInt32())
			m.buddies[i].Group = model.ReadPaddedString(r, 17)
		}
		for i := range m.buddies {
			m.buddies[i].InShop = r.ReadUint32() != 0
		}
	}
}
