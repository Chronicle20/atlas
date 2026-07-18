package interaction

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

type VisitorType byte

const (
	BaseVisitorType     VisitorType = 0
	MerchantVisitorType VisitorType = 2
)

// GameRecord is the 20-byte (5 x int32) mini-game win/tie/loss/points record
// (client sub_4E42FC, DecodeBuffer(0x14)). Used by the game clientbound
// packets (clientbound.InteractionMiniGameRoom / InteractionMiniGameResult);
// game rooms are NOT modelled as Visitors here — see
// clientbound.InteractionMiniGameRoom for the verified room-enter layout.
type GameRecord struct {
	Unknown uint32
	Wins    uint32
	Ties    uint32
	Losses  uint32
	Points  uint32
}

type Visitor struct {
	visitorType  VisitorType
	slot         byte
	avatar       model.Avatar
	name         string
	itemId       uint32
	merchantName string
}

func NewBaseVisitor(slot byte, avatar model.Avatar, name string) Visitor {
	return Visitor{visitorType: BaseVisitorType, slot: slot, avatar: avatar, name: name}
}

func NewMerchantVisitor(itemId uint32, merchantName string) Visitor {
	return Visitor{visitorType: MerchantVisitorType, slot: 0, itemId: itemId, merchantName: merchantName}
}

func (v Visitor) VisitorType() VisitorType { return v.visitorType }
func (v Visitor) Slot() byte               { return v.slot }
func (v Visitor) Avatar() model.Avatar     { return v.avatar }
func (v Visitor) Name() string             { return v.name }
func (v Visitor) ItemId() uint32           { return v.itemId }
func (v Visitor) MerchantName() string     { return v.merchantName }

func (v Visitor) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		switch v.visitorType {
		case BaseVisitorType:
			w.WriteByte(v.slot)
			w.WriteByteArray(v.avatar.Encode(l, ctx)(options))
			w.WriteAsciiString(v.name)
		case MerchantVisitorType:
			w.WriteByte(0)
			w.WriteInt(v.itemId)
			w.WriteAsciiString(v.merchantName)
		}
		return w.Bytes()
	}
}

func (v *Visitor) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		switch v.visitorType {
		case BaseVisitorType:
			v.slot = r.ReadByte()
			v.avatar.Decode(l, ctx)(r, options)
			v.name = r.ReadAsciiString()
		case MerchantVisitorType:
			v.slot = r.ReadByte()
			v.itemId = r.ReadUint32()
			v.merchantName = r.ReadAsciiString()
		}
	}
}

// decodeVisitorForRoom decodes a visitor from the reader based on room type and slot.
func decodeVisitorForRoom(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, roomType byte) *Visitor {
	slot := r.ReadByte()
	if slot == 0xFF {
		return nil
	}

	// NOTE: game rooms (roomType 1/2) never flow through here — their
	// room-enter layout is different (two separate 0xFF-terminated lists) and
	// is modelled by clientbound.InteractionMiniGameRoom.
	v := &Visitor{}
	switch roomType {
	case 5: // Merchant — slot 0 is owner, others are base
		if slot == 0 {
			v.visitorType = MerchantVisitorType
			v.slot = 0
			v.itemId = r.ReadUint32()
			v.merchantName = r.ReadAsciiString()
		} else {
			v.visitorType = BaseVisitorType
			v.slot = slot
			v.avatar.Decode(l, ctx)(r, options)
			v.name = r.ReadAsciiString()
		}
	default: // Trade, PersonalShop, CashTrade — base visitors
		v.visitorType = BaseVisitorType
		v.slot = slot
		v.avatar.Decode(l, ctx)(r, options)
		v.name = r.ReadAsciiString()
	}
	return v
}
