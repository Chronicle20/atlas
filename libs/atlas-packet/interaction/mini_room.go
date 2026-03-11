package interaction

import (
	"context"

	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
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

type MiniRoom interface {
	Type() MiniRoomType
	Is(mrt MiniRoomType) bool
	Capacity() byte
	Visitors() []MiniRoomVisitor
	Spawn(characterId uint32) packet.Encode
	Despawn(characterId uint32) packet.Encode
	Enter(characterId uint32) packet.Encode
}

type MiniRoomVisitor interface {
	Enter() packet.Encode
}

type MiniRoomBase struct {
	MiniRoomTypeVal MiniRoomType
	Id              uint32
	Title           string
	Private         bool
	GameKind        byte
	GameOn          bool
	CapacityVal     byte
	OwnerId         uint32
	VisitorList     []MiniRoomVisitor
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
			w.WriteByte(m.GameKind)
			w.WriteByte(byte(len(m.VisitorList)))
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

func (m *MiniRoomBase) Enter(_ uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			l.Fatalf("concrete implementation needed")
			return []byte{}
		}
	}
}

type GameMiniRoom struct {
	*MiniRoomBase
	Tournament bool
	Round      byte
}

func (m *GameMiniRoom) Spawn(_ uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return []byte{}
		}
	}
}

func (m *GameMiniRoom) Enter(_ uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(m.Type()))
			w.WriteByte(m.Capacity())
			for _, v := range m.Visitors() {
				w.WriteByteArray(v.Enter()(l, ctx)(options))
			}
			w.WriteByte(0xFF)
			w.WriteAsciiString(m.Title)
			w.WriteByte(m.GameKind)
			w.WriteBool(m.Tournament)
			if m.Tournament {
				w.WriteByte(m.Round)
			}
			return w.Bytes()
		}
	}
}

func NewOmokMiniRoom(owner MiniGameRoomVisitor) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &GameMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			MiniRoomTypeVal: OmokMiniRoomType,
			CapacityVal:     2,
			VisitorList:     visitors,
		},
	}
}

func NewMatchCardMiniRoom(owner MiniGameRoomVisitor) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &GameMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			MiniRoomTypeVal: MatchCardMiniRoomType,
			CapacityVal:     2,
			VisitorList:     visitors,
		},
	}
}

func NewTradeMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &MiniRoomBase{
		MiniRoomTypeVal: TradeMiniRoomType,
		CapacityVal:     2,
		VisitorList:     visitors,
	}
}

type PersonalShopMiniRoom struct {
	*MiniRoomBase
	MaxItemCount byte
	Items        []ShopItem
}

func (m *PersonalShopMiniRoom) Enter(_ uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(m.Type()))
			w.WriteByte(m.Capacity())
			for _, v := range m.Visitors() {
				w.WriteByteArray(v.Enter()(l, ctx)(options))
			}
			w.WriteByte(0xFF)
			w.WriteAsciiString(m.Title)
			w.WriteByte(m.MaxItemCount)
			w.WriteByte(byte(len(m.Items)))
			for _, i := range m.Items {
				w.WriteShort(i.PerBundle)
				w.WriteShort(i.Quantity)
				w.WriteInt(i.Price)
				w.WriteByteArray(i.Asset.Encode(l, ctx)(options))
			}
			return w.Bytes()
		}
	}
}

func NewPersonalShopMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &PersonalShopMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			MiniRoomTypeVal: PersonalShopMiniRoomType,
			CapacityVal:     4,
			VisitorList:     visitors,
		},
		MaxItemCount: 16,
	}
}

type ShopItem struct {
	PerBundle uint16
	Quantity  uint16
	Price     uint32
	Asset     model.Asset
}

type MiniRoomMessage struct {
	Message string
	Slot    byte
}

type MerchantShopMiniRoom struct {
	*MiniRoomBase
	OwnerName    string
	Meso         uint32
	MaxItemCount byte
	Messages     []MiniRoomMessage
	Items        []ShopItem
}

func (m *MerchantShopMiniRoom) Enter(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(m.Type()))
			w.WriteByte(m.Capacity())
			for _, v := range m.Visitors() {
				w.WriteByteArray(v.Enter()(l, ctx)(options))
			}
			w.WriteByte(0xFF)
			if characterId == m.OwnerId {
				w.WriteShort(uint16(len(m.Messages)))
				for _, i := range m.Messages {
					w.WriteAsciiString(i.Message)
					w.WriteByte(i.Slot)
				}
			} else {
				w.WriteShort(0)
			}
			w.WriteAsciiString(m.OwnerName)
			w.WriteByte(m.MaxItemCount)
			w.WriteInt(m.Meso)
			w.WriteByte(byte(len(m.Items)))
			for _, i := range m.Items {
				w.WriteShort(i.PerBundle)
				w.WriteShort(i.Quantity)
				w.WriteInt(i.Price)
				w.WriteByteArray(i.Asset.Encode(l, ctx)(options))
			}
			return w.Bytes()
		}
	}
}

func NewMerchantShopMiniRoom(owner MerchantOwnerVisitor) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &MerchantShopMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			MiniRoomTypeVal: MerchantShopMiniRoomType,
			CapacityVal:     4,
			VisitorList:     visitors,
		},
		MaxItemCount: 16,
	}
}

func NewCashTradeMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &MiniRoomBase{
		MiniRoomTypeVal: CashTradeMiniRoomType,
		CapacityVal:     2,
		VisitorList:     visitors,
	}
}

type MiniRoomVisitorBase struct {
	Name   string
	Slot   byte
	Avatar model.Avatar
}

func (m *MiniRoomVisitorBase) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(m.Slot)
			w.WriteByteArray(m.Avatar.Encode(l, ctx)(options))
			w.WriteAsciiString(m.Name)
			return w.Bytes()
		}
	}
}

type MiniGameRoomVisitor struct {
	Mrb    MiniRoomVisitorBase
	Record MiniGameRecord
}

func (m *MiniGameRoomVisitor) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByteArray(m.Mrb.Enter()(l, ctx)(options))
			w.WriteByteArray(m.Record.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}

type MerchantOwnerVisitor struct {
	ItemId       item.Id
	MerchantName string
}

func (m *MerchantOwnerVisitor) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0)
			w.WriteInt(uint32(m.ItemId))
			w.WriteAsciiString(m.MerchantName)
			return w.Bytes()
		}
	}
}

type MiniGameRecord struct {
	Unknown uint32
	Wins    uint32
	Ties    uint32
	Losses  uint32
	Points  uint32
}

func (m *MiniGameRecord) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.Unknown)
		w.WriteInt(m.Wins)
		w.WriteInt(m.Ties)
		w.WriteInt(m.Losses)
		w.WriteInt(m.Points)
		return w.Bytes()
	}
}
