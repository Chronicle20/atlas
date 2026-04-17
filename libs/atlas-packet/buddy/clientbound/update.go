package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyUpdateWriter = "BuddyUpdate"

type Update struct {
	mode          byte
	characterId   uint32
	characterName string
	group         string
	channelId     channel.Id
	inShop        bool
}

func NewBuddyUpdate(mode byte, characterId uint32, characterName string, group string, channelId channel.Id, inShop bool) Update {
	return Update{mode: mode, characterId: characterId, characterName: characterName, group: group, channelId: channelId, inShop: inShop}
}

func (m Update) Mode() byte              { return m.mode }
func (m Update) CharacterId() uint32      { return m.characterId }
func (m Update) CharacterName() string    { return m.characterName }
func (m Update) Group() string            { return m.group }
func (m Update) ChannelId() channel.Id    { return m.channelId }
func (m Update) InShop() bool             { return m.inShop }
func (m Update) Operation() string        { return BuddyUpdateWriter }

func (m Update) String() string {
	return fmt.Sprintf("buddy update characterId [%d]", m.characterId)
}

func (m Update) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.characterId)
		bm := model.Buddy{
			FriendId:    m.characterId,
			FriendName:  m.characterName,
			Flag:        0,
			ChannelId:   m.channelId,
			FriendGroup: m.group,
		}
		w.WriteByteArray(bm.Encode(l, ctx)(options))
		w.WriteBool(m.inShop)
		return w.Bytes()
	}
}

func (m *Update) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.characterId = r.ReadUint32()
		_ = r.ReadUint32()                              // friendId (same as characterId)
		m.characterName = model.ReadPaddedString(r, 13) // friendName
		_ = r.ReadByte()                                // flag
		m.channelId = channel.Id(r.ReadInt32())
		m.group = model.ReadPaddedString(r, 17) // friendGroup
		m.inShop = r.ReadBool()
	}
}
