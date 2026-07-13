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

// RoomSoldItem is one entry of the hired-merchant sale ledger the client reads in
// the customer view (DecodeSoldItemList sub_518EFD @0x518efd): item id, quantity,
// price, and the buyer's name.
type RoomSoldItem struct {
	ItemId    uint32
	Quantity  uint16
	Price     uint32
	BuyerName string
}

type Room struct {
	roomType     RoomType
	capacity     byte
	ownerView    bool
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
	soldItems    []RoomSoldItem
	soldTotal    uint32
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

// NewPersonalShopRoom builds a personal-store (roomType 4) enter-result room.
// ownerView selects the CMiniRoomBaseDlg::OnEnterResultBase second header byte
// (real offset 0xC8): the client branches on it in CPersonalShopDlg::OnEnterResult
// (v83 @0x6fc528 `if(*(this+50))`) — nonzero opens the owner's add-item management
// UI, zero the visitor buy UI. Pass true when the recipient is the shop owner.
func NewPersonalShopRoom(ownerView bool, visitors []Visitor, title string, maxItemCount byte, items []RoomShopItem) Room {
	return Room{
		roomType:     PersonalShopRoomType,
		capacity:     4,
		ownerView:    ownerView,
		visitors:     visitors,
		title:        title,
		maxItemCount: maxItemCount,
		items:        items,
	}
}

// NewMerchantShopRoom builds a hired-merchant (roomType 5) enter-result room.
// ownerView selects the same OnEnterResultBase header byte (offset 0xC8) that the
// client branches on in CEntrustedShopDlg::OnEnterResult (v83 @0x518a7e): the owner
// view skips the meso/flag/sale-ledger block, the customer view reads it. title is
// the shop name the client reads at the common tail (DecodeStr this+105 @0x518c8f) —
// distinct from ownerName (the merchant owner's character name, this+479 @0x518a54).
func NewMerchantShopRoom(ownerView bool, visitors []Visitor, messages []RoomMessage, ownerName string, title string, maxItemCount byte, meso uint32, items []RoomShopItem) Room {
	return Room{
		roomType:     MerchantShopRoomType,
		capacity:     4,
		ownerView:    ownerView,
		visitors:     visitors,
		messages:     messages,
		ownerName:    ownerName,
		title:        title,
		maxItemCount: maxItemCount,
		meso:         meso,
		items:        items,
	}
}

func (r Room) RoomType() RoomType         { return r.roomType }
func (r Room) Capacity() byte             { return r.capacity }
func (r Room) OwnerView() bool            { return r.ownerView }
func (r Room) Visitors() []Visitor        { return r.visitors }

// ownerViewByte encodes the OnEnterResultBase second header byte (offset 0xC8):
// 1 = owner view, 0 = visitor view.
func (r Room) ownerViewByte() byte {
	if r.ownerView {
		return 1
	}
	return 0
}
func (r Room) Title() string              { return r.title }
func (r Room) GameKind() byte             { return r.gameKind }
func (r Room) Tournament() bool           { return r.tournament }
func (r Room) Round() byte                { return r.round }
func (r Room) MaxItemCount() byte         { return r.maxItemCount }
func (r Room) Items() []RoomShopItem      { return r.items }
func (r Room) Messages() []RoomMessage    { return r.messages }
func (r Room) OwnerName() string          { return r.ownerName }
func (r Room) Meso() uint32               { return r.meso }
func (r Room) SoldItems() []RoomSoldItem  { return r.soldItems }
func (r Room) SoldTotal() uint32          { return r.soldTotal }

func (rm Room) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(rm.roomType))
		w.WriteByte(rm.capacity)
		// CMiniRoomBaseDlg::OnEnterResultBase reads a SECOND header byte here
		// (Decode1 -> *(this+0xC8), v83 @0x65ec6b) for EVERY room type, before the
		// visitor loop. Omitting it shifts the whole visitor list and over-reads the
		// tail -> live "error 38". The client branches on it (owner vs visitor view).
		w.WriteByte(rm.ownerViewByte())
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
			// Hired-merchant (roomType 5) enter-result tail — CEntrustedShopDlg::OnEnterResult
			// (v83 @0x518873). Owner announcements: Decode2 count; count x {DecodeStr, Decode1}.
			w.WriteShort(uint16(len(rm.messages)))
			for _, msg := range rm.messages {
				w.WriteAsciiString(msg.Message)
				w.WriteByte(msg.Slot)
			}
			// ownerName: DecodeStr -> this+479 (@0x518a54).
			w.WriteAsciiString(rm.ownerName)
			// Customer branch only (owner byte *(this+0xC8) == 0, @0x518a7e): the shop's
			// meso, a sold-out flag, then the sale-transaction ledger (DecodeSoldItemList
			// sub_518EFD @0x518efd). The owner view skips all of this.
			if !rm.ownerView {
				w.WriteInt(rm.meso)                  // Decode4 this[482] @0x518b04
				w.WriteByte(0)                       // Decode1 flag @0x518b0a (0 = not sold out)
				w.WriteByte(byte(len(rm.soldItems))) // Decode1 count @0x518f1c
				for _, s := range rm.soldItems {
					w.WriteInt(s.ItemId)            // Decode4 @0x518f4a
					w.WriteShort(s.Quantity)        // Decode2 @0x518f69
					w.WriteInt(s.Price)             // Decode4 @0x518f78
					w.WriteAsciiString(s.BuyerName) // DecodeStr @0x518f7f
				}
				w.WriteInt(rm.soldTotal) // Decode4 total @0x518fbc
			}
			// Common tail: title (DecodeStr this+105 @0x518c8f), maxItem (Decode1
			// this+109 @0x518d12).
			w.WriteAsciiString(rm.title)
			w.WriteByte(rm.maxItemCount)
			// OnRefresh (vtable+112 = CEntrustedShopDlg::OnRefresh @0x518852, BOTH views):
			// Decode4 withdrawable meso (this[481] @0x518864), then CPersonalShopDlg::OnRefresh
			// (@0x6fcc4e): Decode1 count; count x {Decode2 perBundle, Decode2 qty, Decode4
			// price, GW_ItemSlotBase}.
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
		rm.ownerView = r.ReadByte() != 0

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
			// Customer branch (owner byte == 0): meso + sold-out flag + sale ledger.
			if !rm.ownerView {
				rm.meso = r.ReadUint32()
				_ = r.ReadByte() // sold-out flag (@0x518b0a)
				soldCount := r.ReadByte()
				rm.soldItems = make([]RoomSoldItem, soldCount)
				for i := byte(0); i < soldCount; i++ {
					rm.soldItems[i].ItemId = r.ReadUint32()
					rm.soldItems[i].Quantity = r.ReadUint16()
					rm.soldItems[i].Price = r.ReadUint32()
					rm.soldItems[i].BuyerName = r.ReadAsciiString()
				}
				rm.soldTotal = r.ReadUint32()
			}
			rm.title = r.ReadAsciiString()
			rm.maxItemCount = r.ReadByte()
			// OnRefresh withdrawable meso (this[481]); Atlas populates both meso slots
			// from the single shop balance, so this overwrites with the same value.
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
