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

// RoomSoldItem is one entry of the hired-merchant sale ledger the client reads in
// the customer view (DecodeSoldItemList sub_518EFD @0x518efd): item id, quantity,
// price, and the buyer's name.
type RoomSoldItem struct {
	ItemId    uint32
	Quantity  uint16
	Price     uint32
	BuyerName string
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
	position     byte
	visitors     []Visitor
	title        string
	maxItemCount byte
	items        []RoomShopItem
	messages     []RoomMessage
	ownerName    string
	meso         uint32
	openTime     uint16
	firstTime    bool
	soldItems    []RoomSoldItem
	soldTotal    uint32
}

// NewPersonalShopRoom builds a personal-store (roomType 4) enter-result room.
// position is the recipient's position in the room — 0 = owner, 1..3 = the
// visitor's slot. It lands in the CMiniRoomBaseDlg::OnEnterResultBase second
// header byte (v83 @0x65ec6b -> *(this+0xC8)); CPersonalShopDlg::OnEnterResult
// branches on it (v83 @0x6fc528 `if(*(this+50))`): ZERO opens the owner's
// add-item management UI, nonzero the visitor buy UI.
func NewPersonalShopRoom(position byte, visitors []Visitor, title string, maxItemCount byte, items []RoomShopItem) Room {
	return Room{
		roomType:     PersonalShopRoomType,
		capacity:     4,
		position:     position,
		visitors:     visitors,
		title:        title,
		maxItemCount: maxItemCount,
		items:        items,
	}
}

// NewMerchantShopRoom builds a hired-merchant (roomType 5) enter-result room.
// position is the recipient's position in the room — 0 = owner, 1..3 = the
// visitor's slot (same OnEnterResultBase header byte, offset 0xC8).
// CEntrustedShopDlg::OnEnterResult branches on it (v83 @0x518a7e): the
// position==0 (owner) view decodes the extra open-time/first-time/sale-ledger
// block and opens the owner management UI (CWvsContext::UI_Open gated on
// `!this[50]` @0x518d3d); a visitor view skips that block. Owner rooms must
// also call SetOwnerLedger to populate the owner-only block. title is the shop
// name the client reads at the common tail (DecodeStr this+105 @0x518c8f) —
// distinct from ownerName (the merchant owner's character name, this+479
// @0x518a54).
func NewMerchantShopRoom(position byte, visitors []Visitor, messages []RoomMessage, ownerName string, title string, maxItemCount byte, meso uint32, items []RoomShopItem) Room {
	return Room{
		roomType:     MerchantShopRoomType,
		capacity:     4,
		position:     position,
		visitors:     visitors,
		messages:     messages,
		ownerName:    ownerName,
		title:        title,
		maxItemCount: maxItemCount,
		meso:         meso,
		items:        items,
	}
}

// SetOwnerLedger populates the owner-only (position 0) block of a
// hired-merchant room: minutes the shop has been open (the client's Decode4
// @0x518b04 reads a packed int — low short always 0, high short the
// minutes), whether this is the first (creation-time) view of the shop, the
// sale-transaction ledger, and the merchant's accrued meso total that
// terminates the ledger (sub_518EFD @0x518fbc).
func (r Room) SetOwnerLedger(openTime uint16, firstTime bool, soldItems []RoomSoldItem, ledgerTotal uint32) Room {
	r.openTime = openTime
	r.firstTime = firstTime
	r.soldItems = soldItems
	r.soldTotal = ledgerTotal
	return r
}

func (r Room) RoomType() RoomType         { return r.roomType }
func (r Room) Capacity() byte             { return r.capacity }
func (r Room) Position() byte             { return r.position }
func (r Room) OpenTime() uint16           { return r.openTime }
func (r Room) FirstTime() bool            { return r.firstTime }
func (r Room) Visitors() []Visitor        { return r.visitors }
func (r Room) Title() string              { return r.title }
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
		// tail -> live "error 38". It is the recipient's position in the room:
		// 0 = owner, 1..3 = visitor slot; the shop dialogs branch owner/visitor
		// view on it (personal @0x6fc528, entrusted @0x518a7e).
		w.WriteByte(rm.position)
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
				// nNumber (quantity) then nSet (perBundle) — see OnRefresh note below.
				w.WriteShort(item.Quantity)
				w.WriteShort(item.PerBundle)
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
			// OWNER branch only (position byte *(this+0xC8) == 0, @0x518a7e): packed
			// open-time (Decode4 this[482] @0x518b04 — low short always 0, high
			// short the minutes), first-time flag (Decode1 @0x518b0a — branches the
			// owner UI), then the sale-transaction ledger (DecodeSoldItemList
			// sub_518EFD @0x518efd) terminated by the accrued meso total. Visitor
			// views skip all of this — the client decodes it only in the
			// position==0 branch.
			if rm.position == 0 {
				w.WriteShort(0)                      // Decode4 low short @0x518b04 (always 0)
				w.WriteShort(rm.openTime)            // Decode4 high short (minutes open)
				w.WriteBool(rm.firstTime)            // Decode1 @0x518b0a
				w.WriteByte(byte(len(rm.soldItems))) // Decode1 count @0x518f1c
				for _, s := range rm.soldItems {
					w.WriteInt(s.ItemId)            // Decode4 @0x518f4a
					w.WriteShort(s.Quantity)        // Decode2 @0x518f69
					w.WriteInt(s.Price)             // Decode4 @0x518f78
					w.WriteAsciiString(s.BuyerName) // DecodeStr @0x518f7f
				}
				w.WriteInt(rm.soldTotal) // Decode4 accrued meso total @0x518fbc
			}
			// Common tail: title (DecodeStr this+105 @0x518c8f), maxItem (Decode1
			// this+109 @0x518d12).
			w.WriteAsciiString(rm.title)
			w.WriteByte(rm.maxItemCount)
			// OnRefresh (vtable+112 = CEntrustedShopDlg::OnRefresh @0x518852, BOTH views):
			// Decode4 withdrawable meso (this[481] @0x518864), then CPersonalShopDlg::OnRefresh
			// (@0x6fcc4e): Decode1 count; count x {Decode2 nNumber, Decode2 nSet, Decode4
			// price, GW_ItemSlotBase}. v95 PDB names offset0=nNumber (quantity),
			// offset4=nSet (per-bundle) — so quantity is written first (task-127).
			w.WriteInt(rm.meso)
			w.WriteByte(byte(len(rm.items)))
			for _, item := range rm.items {
				w.WriteShort(item.Quantity)
				w.WriteShort(item.PerBundle)
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
		rm.position = r.ReadByte()

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
				rm.items[i].Quantity = r.ReadUint16()
				rm.items[i].PerBundle = r.ReadUint16()
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
			// Owner branch (position byte == 0): packed open-time + first-time flag
			// + sale ledger terminated by the accrued meso total.
			if rm.position == 0 {
				_ = r.ReadUint16() // Decode4 low short (always 0)
				rm.openTime = r.ReadUint16()
				rm.firstTime = r.ReadByte() != 0
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
				rm.items[i].Quantity = r.ReadUint16()
				rm.items[i].PerBundle = r.ReadUint16()
				rm.items[i].Price = r.ReadUint32()
				rm.items[i].Asset = model.NewAsset(true, 0, 0, time.Time{})
				rm.items[i].Asset.Decode(l, ctx)(r, options)
			}
		}
	}
}
