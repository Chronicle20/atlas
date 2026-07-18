package interaction

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MiniRoomWriter = "MiniRoom"

type MiniRoomType byte

const (
	OmokMiniRoomType         MiniRoomType = 1 // COmokDlg
	MatchCardMiniRoomType    MiniRoomType = 2 // CMemoryGameDlg
	TradeMiniRoomType        MiniRoomType = 3 // CTradingRoomDlg
	PersonalShopMiniRoomType MiniRoomType = 4 // CPersonalShopDlg
	MerchantShopMiniRoomType MiniRoomType = 5 // CEntrustedShopDlg
	CashTradeMiniRoomType    MiniRoomType = 6 // CCashTradingRoomDlg
)

type MiniRoomVisitor interface {
	Enter() packet.Encode
}

// MiniRoomBase is the field-level mini-room balloon that attaches to a player's
// avatar (UPDATE_CHAR_BOX / CUser::OnMiniRoomBalloon). The full room interior is
// encoded separately by the Room type (room.go); only Spawn/Despawn are on the
// wire for the avatar box.
type MiniRoomBase struct {
	MiniRoomTypeVal MiniRoomType
	Id              uint32
	Title           string
	Private         bool
	// Spec is the balloon's nSpec byte (CUser::OnMiniRoomBalloon reads it as the
	// 5th Decode1 and passes it to CChatBalloon::MakeMiniRoomBalloon as nSpec).
	// Its meaning is per room type: for a personal shop (type 4) it is the
	// store-sign skin index (WZ .../PSSkin/<Spec>); for a game room it is the
	// game kind. Left 0 for the plain personal-store sign.
	Spec         byte
	GameOn       bool
	CapacityVal  byte
	OwnerId      uint32
	VisitorCount byte
	VisitorList  []MiniRoomVisitor
}

func (m *MiniRoomBase) Type() MiniRoomType {
	return m.MiniRoomTypeVal
}

func (m *MiniRoomBase) Is(miniRoomType MiniRoomType) bool {
	return m.MiniRoomTypeVal == miniRoomType
}

func (m *MiniRoomBase) Capacity() byte {
	return m.CapacityVal
}

func (m *MiniRoomBase) Visitors() []MiniRoomVisitor {
	return m.VisitorList
}

func (m *MiniRoomBase) Spawn(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(byte(m.Type()))
			w.WriteInt(m.Id)
			w.WriteAsciiString(m.Title)
			w.WriteBool(m.Private)
			w.WriteByte(m.Spec)
			w.WriteByte(m.VisitorCount)
			w.WriteByte(m.CapacityVal)
			w.WriteBool(m.GameOn)
			return w.Bytes()
		}
	}
}

func (m *MiniRoomBase) Despawn(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}
