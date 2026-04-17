package interaction

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type RoomType byte

const (
	OmokRoomType         RoomType = 1
	MatchCardRoomType    RoomType = 2
	TradeRoomType        RoomType = 3
	PersonalShopRoomType RoomType = 4
	MerchantShopRoomType RoomType = 5
	CashTradeRoomType    RoomType = 6
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

type Room struct {
	roomType     RoomType
	capacity     byte
	visitors     []Visitor
	title        string
	gameKind     byte
	tournament   bool
	round        byte
	maxItemCount byte
	items        []RoomShopItem
	messages     []RoomMessage
	ownerName    string
	meso         uint32
}

func NewGameRoom(roomType RoomType, capacity byte, visitors []Visitor, title string, gameKind byte, tournament bool, round byte) Room {
	return Room{
		roomType:   roomType,
		capacity:   capacity,
		visitors:   visitors,
		title:      title,
		gameKind:   gameKind,
		tournament: tournament,
		round:      round,
	}
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
func (r Room) GameKind() byte             { return r.gameKind }
func (r Room) Tournament() bool           { return r.tournament }
func (r Room) Round() byte                { return r.round }
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
		case OmokRoomType, MatchCardRoomType:
			w.WriteAsciiString(rm.title)
			w.WriteByte(rm.gameKind)
			w.WriteBool(rm.tournament)
			if rm.tournament {
				w.WriteByte(rm.round)
			}
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
		case OmokRoomType, MatchCardRoomType:
			rm.title = r.ReadAsciiString()
			rm.gameKind = r.ReadByte()
			rm.tournament = r.ReadBool()
			if rm.tournament {
				rm.round = r.ReadByte()
			}
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
