package model

import (
	"atlas-channel/asset"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

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
	ToPacketRoom(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, characterId uint32) interactionpkt.Room
}

type MiniRoomVisitor interface {
	Enter() packet.Encode
	ToPacketVisitor(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}) interactionpkt.Visitor
}

type MiniRoomBase struct {
	miniRoomType MiniRoomType
	id           uint32
	title        string
	private      bool
	gameKind     byte
	gameOn       bool
	capacity     byte
	ownerId      uint32
	visitors     []MiniRoomVisitor
}

func (m *MiniRoomBase) Type() MiniRoomType {
	return m.miniRoomType
}

func (m *MiniRoomBase) Is(miniRoomType MiniRoomType) bool {
	return m.miniRoomType == miniRoomType
}

func (m *MiniRoomBase) Capacity() byte {
	return m.capacity
}

func (m *MiniRoomBase) Visitors() []MiniRoomVisitor {
	return m.visitors
}

func (m *MiniRoomBase) Spawn(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(byte(m.Type()))
			w.WriteInt(m.id)
			w.WriteAsciiString(m.title)
			w.WriteBool(m.private)
			w.WriteByte(m.gameKind)
			w.WriteByte(byte(len(m.visitors)))
			w.WriteByte(m.capacity)
			w.WriteBool(m.gameOn)
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

func (m *MiniRoomBase) ToPacketRoom(_ logrus.FieldLogger, _ context.Context, _ map[string]interface{}, _ uint32) interactionpkt.Room {
	panic("concrete ToPacketRoom implementation needed")
}

type GameMiniRoom struct {
	*MiniRoomBase
	tournament bool
	round      byte
}

func (m *GameMiniRoom) Spawn(_ uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return []byte{}
		}
	}
}

func (m *GameMiniRoom) ToPacketRoom(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, _ uint32) interactionpkt.Room {
	visitors := make([]interactionpkt.Visitor, 0, len(m.Visitors()))
	for _, v := range m.Visitors() {
		visitors = append(visitors, v.ToPacketVisitor(l, ctx, options))
	}
	return interactionpkt.NewGameRoom(interactionpkt.RoomType(m.Type()), m.Capacity(), visitors, m.title, m.gameKind, m.tournament, m.round)
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
			w.WriteAsciiString(m.title)
			w.WriteByte(m.gameKind)
			w.WriteBool(m.tournament)
			if m.tournament {
				w.WriteByte(m.round)
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
			miniRoomType: OmokMiniRoomType,
			capacity:     2,
			visitors:     visitors,
		},
	}
}

func NewMatchCardMiniRoom(owner MiniGameRoomVisitor) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &GameMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			miniRoomType: MatchCardMiniRoomType,
			capacity:     2,
			visitors:     visitors,
		},
	}
}

func NewTradeMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &MiniRoomBase{
		miniRoomType: TradeMiniRoomType,
		capacity:     2,
		visitors:     visitors,
	}
}

type PersonalShopMiniRoom struct {
	*MiniRoomBase
	maxItemCount byte
	items        []ShopItem
}

func (m *PersonalShopMiniRoom) ToPacketRoom(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, _ uint32) interactionpkt.Room {
	visitors := make([]interactionpkt.Visitor, 0, len(m.Visitors()))
	for _, v := range m.Visitors() {
		visitors = append(visitors, v.ToPacketVisitor(l, ctx, options))
	}
	items := make([]interactionpkt.RoomShopItem, 0, len(m.items))
	for _, i := range m.items {
		items = append(items, interactionpkt.RoomShopItem{
			PerBundle: i.perBundle,
			Quantity:  i.quantity,
			Price:     i.price,
			Asset:     NewAsset(true, i.asset),
		})
	}
	return interactionpkt.NewPersonalShopRoom(visitors, m.title, m.maxItemCount, items)
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
			w.WriteAsciiString(m.title)
			w.WriteByte(m.maxItemCount)
			w.WriteByte(byte(len(m.items)))
			for _, i := range m.items {
				w.WriteShort(i.perBundle)
				w.WriteShort(i.quantity)
				w.WriteInt(i.price)
				_ = NewAssetWriter(l, ctx, options, w)(true)(i.asset)
			}
			return w.Bytes()
		}
	}
}

type MiniRoomMessage struct {
	message string
	slot    byte
}

type ShopItem struct {
	perBundle uint16
	quantity  uint16
	price     uint32
	asset     asset.Model
}

func NewPersonalShopMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &PersonalShopMiniRoom{
		MiniRoomBase: &MiniRoomBase{
			miniRoomType: PersonalShopMiniRoomType,
			capacity:     4,
			visitors:     visitors,
		},
		maxItemCount: 16,
	}
}

type MerchantShopMiniRoom struct {
	*MiniRoomBase
	ownerName    string
	meso         uint32
	maxItemCount byte
	messages     []MiniRoomMessage
	items        []ShopItem
}

func (m *MerchantShopMiniRoom) ToPacketRoom(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, characterId uint32) interactionpkt.Room {
	visitors := make([]interactionpkt.Visitor, 0, len(m.Visitors()))
	for _, v := range m.Visitors() {
		visitors = append(visitors, v.ToPacketVisitor(l, ctx, options))
	}
	var messages []interactionpkt.RoomMessage
	if characterId == m.ownerId {
		for _, msg := range m.messages {
			messages = append(messages, interactionpkt.RoomMessage{Message: msg.message, Slot: msg.slot})
		}
	}
	items := make([]interactionpkt.RoomShopItem, 0, len(m.items))
	for _, i := range m.items {
		items = append(items, interactionpkt.RoomShopItem{
			PerBundle: i.perBundle,
			Quantity:  i.quantity,
			Price:     i.price,
			Asset:     NewAsset(true, i.asset),
		})
	}
	return interactionpkt.NewMerchantShopRoom(visitors, messages, m.ownerName, m.maxItemCount, m.meso, items)
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
			if characterId == m.ownerId {
				w.WriteShort(uint16(len(m.messages)))
				for _, i := range m.messages {
					w.WriteAsciiString(i.message)
					w.WriteByte(i.slot)
				}
			} else {
				w.WriteShort(0)
			}
			w.WriteAsciiString(m.ownerName)
			w.WriteByte(m.maxItemCount)
			w.WriteInt(m.meso)
			w.WriteByte(byte(len(m.items)))
			for _, i := range m.items {
				w.WriteShort(i.perBundle)
				w.WriteShort(i.quantity)
				w.WriteInt(i.price)
				_ = NewAssetWriter(l, ctx, options, w)(true)(i.asset)
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
			miniRoomType: MerchantShopMiniRoomType,
			capacity:     4,
			visitors:     visitors,
		},
		maxItemCount: 16,
	}
}

func NewCashTradeMiniRoom(owner MiniRoomVisitorBase) MiniRoom {
	visitors := make([]MiniRoomVisitor, 0)
	visitors = append(visitors, &owner)
	return &MiniRoomBase{
		miniRoomType: CashTradeMiniRoomType,
		capacity:     2,
		visitors:     visitors,
	}
}

type MiniRoomVisitorBase struct {
	name   string
	slot   byte
	avatar packetmodel.Avatar
}

func (m *MiniRoomVisitorBase) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(m.slot)
			w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
			w.WriteAsciiString(m.name)
			return w.Bytes()
		}
	}
}

func (m *MiniRoomVisitorBase) ToPacketVisitor(_ logrus.FieldLogger, _ context.Context, _ map[string]interface{}) interactionpkt.Visitor {
	return interactionpkt.NewBaseVisitor(m.slot, m.avatar, m.name)
}

type MiniGameRoomVisitor struct {
	mrb    MiniRoomVisitorBase
	record interactionpkt.MiniGameRecord
}

func (m *MiniGameRoomVisitor) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByteArray(m.mrb.Enter()(l, ctx)(options))
			w.WriteByteArray(m.record.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}

func (m *MiniGameRoomVisitor) ToPacketVisitor(_ logrus.FieldLogger, _ context.Context, _ map[string]interface{}) interactionpkt.Visitor {
	record := interactionpkt.GameRecord{
		Unknown: m.record.Unknown,
		Wins:    m.record.Wins,
		Ties:    m.record.Ties,
		Losses:  m.record.Losses,
		Points:  m.record.Points,
	}
	return interactionpkt.NewGameVisitor(m.mrb.slot, m.mrb.avatar, m.mrb.name, record)
}

type MerchantOwnerVisitor struct {
	itemId       item.Id
	merchantName string
}

func (m *MerchantOwnerVisitor) Enter() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0)
			w.WriteInt(uint32(m.itemId))
			w.WriteAsciiString(m.merchantName)
			return w.Bytes()
		}
	}
}

func (m *MerchantOwnerVisitor) ToPacketVisitor(_ logrus.FieldLogger, _ context.Context, _ map[string]interface{}) interactionpkt.Visitor {
	return interactionpkt.NewMerchantVisitor(uint32(m.itemId), m.merchantName)
}
