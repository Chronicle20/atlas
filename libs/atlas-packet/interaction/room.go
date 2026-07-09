package interaction

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type RoomType byte

// The room-type discriminator bytes are single-sourced in
// libs/atlas-constants/miniroom; RoomType keeps its own packet-domain type but
// derives its values from those shared constants (CLAUDE.md: straightforward
// move over type-alias re-export).
const (
	OmokRoomType         RoomType = RoomType(miniroom.Omok)
	MatchCardRoomType    RoomType = RoomType(miniroom.MatchCards)
	TradeRoomType        RoomType = RoomType(miniroom.Trade)
	PersonalShopRoomType RoomType = RoomType(miniroom.PersonalShop)
	MerchantShopRoomType RoomType = RoomType(miniroom.MerchantShop)
	CashTradeRoomType    RoomType = RoomType(miniroom.CashTrade)
)

type RoomMessage struct {
	Message string
	Slot    byte
}

type RoomShopItem struct {
	PerBundle uint16
	Quantity  uint16
	Price     uint32
	Asset     model.Asset
}

// Room models the shop-family EnterResultSuccess bodies (personal shop /
// hired merchant). Game rooms (Omok / Match Cards) are NOT modelled here:
// their room-enter blob has a different layout (yourSlot byte after capacity;
// avatars and 20-byte records in two SEPARATE 0xFF-terminated lists) and
// lives in clientbound.InteractionMiniGameRoom (IDA-derived; ida-notes.md §G5
// "Room-enter blob — FULL RESOLUTION").
type Room struct {
	roomType     RoomType
	capacity     byte
	visitors     []Visitor
	title        string
	maxItemCount byte
	items        []RoomShopItem
	messages     []RoomMessage
	ownerName    string
	meso         uint32
}

func NewPersonalShopRoom(visitors []Visitor, title string, maxItemCount byte, items []RoomShopItem) Room {
	return Room{
		roomType:     PersonalShopRoomType,
		capacity:     4,
		visitors:     visitors,
		title:        title,
		maxItemCount: maxItemCount,
		items:        items,
	}
}

func NewMerchantShopRoom(visitors []Visitor, messages []RoomMessage, ownerName string, maxItemCount byte, meso uint32, items []RoomShopItem) Room {
	return Room{
		roomType:     MerchantShopRoomType,
		capacity:     4,
		visitors:     visitors,
		messages:     messages,
		ownerName:    ownerName,
		maxItemCount: maxItemCount,
		meso:         meso,
		items:        items,
	}
}

func (r Room) RoomType() RoomType         { return r.roomType }
func (r Room) Capacity() byte             { return r.capacity }
func (r Room) Visitors() []Visitor        { return r.visitors }
func (r Room) Title() string              { return r.title }
func (r Room) MaxItemCount() byte         { return r.maxItemCount }
func (r Room) Items() []RoomShopItem      { return r.items }
func (r Room) Messages() []RoomMessage    { return r.messages }
func (r Room) OwnerName() string          { return r.ownerName }
func (r Room) Meso() uint32               { return r.meso }

func (rm Room) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(rm.roomType))
		w.WriteByte(rm.capacity)
		for _, v := range rm.visitors {
			w.WriteByteArray(v.Encode(l, ctx)(options))
		}
		w.WriteByte(0xFF)

		switch rm.roomType {
		case PersonalShopRoomType:
			w.WriteAsciiString(rm.title)
			w.WriteByte(rm.maxItemCount)
			w.WriteByte(byte(len(rm.items)))
			for _, item := range rm.items {
				w.WriteShort(item.PerBundle)
				w.WriteShort(item.Quantity)
				w.WriteInt(item.Price)
				w.WriteByteArray(item.Asset.Encode(l, ctx)(options))
			}
		case MerchantShopRoomType:
			w.WriteShort(uint16(len(rm.messages)))
			for _, msg := range rm.messages {
				w.WriteAsciiString(msg.Message)
				w.WriteByte(msg.Slot)
			}
			w.WriteAsciiString(rm.ownerName)
			w.WriteByte(rm.maxItemCount)
			w.WriteInt(rm.meso)
			w.WriteByte(byte(len(rm.items)))
			for _, item := range rm.items {
				w.WriteShort(item.PerBundle)
				w.WriteShort(item.Quantity)
				w.WriteInt(item.Price)
				w.WriteByteArray(item.Asset.Encode(l, ctx)(options))
			}
		}
		return w.Bytes()
	}
}

func (rm *Room) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		rm.roomType = RoomType(r.ReadByte())
		rm.capacity = r.ReadByte()

		rm.visitors = nil
		for {
			v := decodeVisitorForRoom(l, ctx, r, options, byte(rm.roomType))
			if v == nil {
				break
			}
			rm.visitors = append(rm.visitors, *v)
		}

		switch rm.roomType {
		case PersonalShopRoomType:
			rm.title = r.ReadAsciiString()
			rm.maxItemCount = r.ReadByte()
			itemCount := r.ReadByte()
			rm.items = make([]RoomShopItem, itemCount)
			for i := byte(0); i < itemCount; i++ {
				rm.items[i].PerBundle = r.ReadUint16()
				rm.items[i].Quantity = r.ReadUint16()
				rm.items[i].Price = r.ReadUint32()
				rm.items[i].Asset = model.NewAsset(true, 0, 0, time.Time{})
				rm.items[i].Asset.Decode(l, ctx)(r, options)
			}
		case MerchantShopRoomType:
			msgCount := r.ReadUint16()
			rm.messages = make([]RoomMessage, msgCount)
			for i := uint16(0); i < msgCount; i++ {
				rm.messages[i].Message = r.ReadAsciiString()
				rm.messages[i].Slot = r.ReadByte()
			}
			rm.ownerName = r.ReadAsciiString()
			rm.maxItemCount = r.ReadByte()
			rm.meso = r.ReadUint32()
			itemCount := r.ReadByte()
			rm.items = make([]RoomShopItem, itemCount)
			for i := byte(0); i < itemCount; i++ {
				rm.items[i].PerBundle = r.ReadUint16()
				rm.items[i].Quantity = r.ReadUint16()
				rm.items[i].Price = r.ReadUint32()
				rm.items[i].Asset = model.NewAsset(true, 0, 0, time.Time{})
				rm.items[i].Asset.Decode(l, ctx)(r, options)
			}
		}
	}
}
