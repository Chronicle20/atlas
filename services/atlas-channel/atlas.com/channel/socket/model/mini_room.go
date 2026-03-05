package model

import (
	"atlas-channel/asset"
	"atlas-channel/character"

	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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
	Is(MiniRoomType) bool
	Capacity() byte
	Visitors() []MiniRoomVisitor
	Enter(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer)
}

type MiniRoomVisitor interface {
	Enter(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer)
}

type MiniRoomBase struct {
	miniRoomType MiniRoomType
	capacity     byte
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

func (m *MiniRoomBase) Enter(l logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		l.Fatalf("concrete implementation needed")
	}
}

type GameMiniRoom struct {
	*MiniRoomBase
	description string
	gameKind    byte
	tournament  bool
	round       byte
}

func (m *GameMiniRoom) Enter(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(m.Type()))
		w.WriteByte(m.Capacity())
		for _, v := range m.Visitors() {
			v.Enter(l, t, options)
		}
		w.WriteByte(-1)
		w.WriteAsciiString(m.description)
		w.WriteByte(m.gameKind)
		w.WriteBool(m.tournament)
		if m.tournament {
			w.WriteByte(m.round)
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
	description  string
	maxItemCount byte
	items        []ShopItem
}

func (m *PersonalShopMiniRoom) Enter(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(m.Type()))
		w.WriteByte(m.Capacity())
		for _, v := range m.Visitors() {
			v.Enter(l, t, options)
		}
		w.WriteByte(-1)
		w.WriteAsciiString(m.description)
		w.WriteByte(m.maxItemCount)
		w.WriteByte(byte(len(m.items)))
		for _, i := range m.items {
			w.WriteShort(i.perBundle)
			w.WriteShort(i.quantity)
			w.WriteInt(i.price)
			_ = NewAssetWriter(l, t, options, w)(true)(i.asset)
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
	description  string
	maxItemCount byte
	messages     []MiniRoomMessage
	items        []ShopItem
}

func (m *MerchantShopMiniRoom) Enter(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(m.Type()))
		w.WriteByte(m.Capacity())
		for _, v := range m.Visitors() {
			v.Enter(l, t, options)
		}
		w.WriteByte(-1)
		// todo only if owner
		w.WriteShort(uint16(len(m.messages)))
		for _, i := range m.messages {
			w.WriteAsciiString(i.message)
			w.WriteByte(i.slot)
		}
		w.WriteAsciiString(m.ownerName)
		w.WriteByte(m.maxItemCount)
		w.WriteInt(m.meso)
		w.WriteByte(byte(len(m.items)))
		for _, i := range m.items {
			w.WriteShort(i.perBundle)
			w.WriteShort(i.quantity)
			w.WriteInt(i.price)
			_ = NewAssetWriter(l, t, options, w)(true)(i.asset)
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
	slot      byte
	character character.Model
}

func (m *MiniRoomVisitorBase) Enter(t logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(m.slot)
		WriteCharacterLook(t)(w, m.character, false)
		w.WriteAsciiString(m.character.Name())
	}
}

type MiniGameRoomVisitor struct {
	mrb    MiniRoomVisitorBase
	record MiniGameRecord
}

func (m *MiniGameRoomVisitor) Enter(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.mrb.Enter(l, t, options)(w)
		m.record.Encode(l, t, options)(w)
	}
}

type MerchantOwnerVisitor struct {
	itemId       item.Id
	merchantName string
}

func (m *MerchantOwnerVisitor) Enter(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(0)
		w.WriteInt(uint32(m.itemId))
		w.WriteAsciiString(m.merchantName)
	}
}
