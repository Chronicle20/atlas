package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterInteractionWriter = "CharacterInteraction"

// InteractionInvite - invite to a mini room
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#Invite
type InteractionInvite struct {
	mode     byte
	roomType byte
	name     string
	dwSN     uint32
}

func NewInteractionInvite(mode byte, roomType byte, name string, dwSN uint32) InteractionInvite {
	return InteractionInvite{mode: mode, roomType: roomType, name: name, dwSN: dwSN}
}

func (m InteractionInvite) Operation() string { return CharacterInteractionWriter }
func (m InteractionInvite) String() string {
	return fmt.Sprintf("invite name [%s] roomType [%d]", m.name, m.roomType)
}

func (m InteractionInvite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.roomType)
		w.WriteAsciiString(m.name)
		w.WriteInt(m.dwSN)
		return w.Bytes()
	}
}

func (m *InteractionInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.roomType = r.ReadByte()
		m.name = r.ReadAsciiString()
		m.dwSN = r.ReadUint32()
	}
}

// InteractionInviteResult - invite result
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#InviteResult
type InteractionInviteResult struct {
	mode    byte
	result  byte
	message string
}

func NewInteractionInviteResult(mode byte, result byte, message string) InteractionInviteResult {
	return InteractionInviteResult{mode: mode, result: result, message: message}
}

func (m InteractionInviteResult) Operation() string { return CharacterInteractionWriter }
func (m InteractionInviteResult) String() string {
	return fmt.Sprintf("invite result [%d]", m.result)
}

func (m InteractionInviteResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.result)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *InteractionInviteResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.result = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}

// InteractionEnter - visitor entering a room
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#Enter
type InteractionEnter struct {
	mode    byte
	visitor interaction.Visitor
}

func NewInteractionEnter(mode byte, visitor interaction.Visitor) InteractionEnter {
	return InteractionEnter{mode: mode, visitor: visitor}
}

func (m InteractionEnter) Operation() string            { return CharacterInteractionWriter }
func (m InteractionEnter) String() string               { return "enter" }
func (m InteractionEnter) Visitor() interaction.Visitor { return m.visitor }

func (m InteractionEnter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.visitor.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *InteractionEnter) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.visitor.Decode(l, ctx)(r, options)
	}
}

// InteractionEnterResultSuccess - successful room entry
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccess
type InteractionEnterResultSuccess struct {
	mode byte
	room interaction.Room
}

func NewInteractionEnterResultSuccess(mode byte, room interaction.Room) InteractionEnterResultSuccess {
	return InteractionEnterResultSuccess{mode: mode, room: room}
}

func (m InteractionEnterResultSuccess) Operation() string      { return CharacterInteractionWriter }
func (m InteractionEnterResultSuccess) String() string         { return "enter result success" }
func (m InteractionEnterResultSuccess) Room() interaction.Room { return m.room }

func (m InteractionEnterResultSuccess) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.room.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *InteractionEnterResultSuccess) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.room.Decode(l, ctx)(r, options)
	}
}

// InteractionChat - chat message in a mini room
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#Chat
type InteractionChat struct {
	mode     byte
	chatType byte
	slot     byte
	message  string
}

func NewInteractionChat(mode byte, chatType byte, slot byte, message string) InteractionChat {
	return InteractionChat{mode: mode, chatType: chatType, slot: slot, message: message}
}

func (m InteractionChat) Operation() string { return CharacterInteractionWriter }
func (m InteractionChat) String() string {
	return fmt.Sprintf("chat slot [%d] message [%s]", m.slot, m.message)
}

func (m InteractionChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.chatType)
		w.WriteByte(m.slot)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *InteractionChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.chatType = r.ReadByte()
		m.slot = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}

// InteractionEnterResultError - failed room entry
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#EnterResultError
type InteractionEnterResultError struct {
	mode      byte
	errorCode byte
}

func NewInteractionEnterResultError(mode byte, errorCode byte) InteractionEnterResultError {
	return InteractionEnterResultError{mode: mode, errorCode: errorCode}
}

func (m InteractionEnterResultError) Operation() string { return CharacterInteractionWriter }
func (m InteractionEnterResultError) String() string {
	return fmt.Sprintf("enter result error [%d]", m.errorCode)
}

func (m InteractionEnterResultError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(0)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *InteractionEnterResultError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// InteractionLeave - visitor leaving a room
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#Leave
type InteractionLeave struct {
	mode   byte
	slot   byte
	status byte
}

func NewInteractionLeave(mode byte, slot byte, status byte) InteractionLeave {
	return InteractionLeave{mode: mode, slot: slot, status: status}
}

func (m InteractionLeave) Operation() string { return CharacterInteractionWriter }
func (m InteractionLeave) String() string {
	return fmt.Sprintf("leave slot [%d] status [%d]", m.slot, m.status)
}

func (m InteractionLeave) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.slot)
		w.WriteByte(m.status)
		return w.Bytes()
	}
}

func (m *InteractionLeave) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slot = r.ReadByte()
		m.status = r.ReadByte()
	}
}

// InteractionUpdateMerchant - refresh shop listings for viewers (hired merchant).
// Wire shape: mode(25) + Decode4 meso + Decode1 count + count x {short perBundle,
// short quantity, int price, GW_ItemSlotBase asset}. The OnPacketBase default case
// virtual-dispatches into CEntrustedShopDlg::OnRefresh (v95 0x51cc30), which reads
// the meso (m_nMoney) then chains CPersonalShopDlg::OnRefresh for the item loop.
// packet-audit:fname CEntrustedShopDlg::OnRefresh#UpdateMerchant
type InteractionUpdateMerchant struct {
	mode     byte
	meso     uint32
	omitMeso bool
	items    []interaction.RoomShopItem
}

func NewInteractionUpdateMerchant(mode byte, meso uint32, items []interaction.RoomShopItem) InteractionUpdateMerchant {
	return InteractionUpdateMerchant{mode: mode, meso: meso, items: items}
}

// NewInteractionUpdatePersonalShop builds the mode-25 refresh for a personal
// shop (item 514). CPersonalShopDlg::OnRefresh reads the item loop directly —
// only the hired-merchant override CEntrustedShopDlg::OnRefresh prefixes the
// Decode4 meso (m_nMoney). Sending the meso to a personal shop makes the client
// read its first byte as the item count (→ 0 items shown), so it is omitted.
func NewInteractionUpdatePersonalShop(mode byte, items []interaction.RoomShopItem) InteractionUpdateMerchant {
	return InteractionUpdateMerchant{mode: mode, omitMeso: true, items: items}
}

func (m InteractionUpdateMerchant) Operation() string { return CharacterInteractionWriter }
func (m InteractionUpdateMerchant) String() string {
	return fmt.Sprintf("update merchant meso [%d] items [%d]", m.meso, len(m.items))
}

func (m InteractionUpdateMerchant) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if !m.omitMeso {
			w.WriteInt(m.meso)
		}
		w.WriteByte(byte(len(m.items)))
		for _, item := range m.items {
			w.WriteShort(item.PerBundle)
			w.WriteShort(item.Quantity)
			w.WriteInt(item.Price)
			w.WriteByteArray(item.Asset.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *InteractionUpdateMerchant) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		if !m.omitMeso {
			m.meso = r.ReadUint32()
		}
		count := int(r.ReadByte())
		m.items = make([]interaction.RoomShopItem, 0, count)
		for i := 0; i < count; i++ {
			var item interaction.RoomShopItem
			item.PerBundle = r.ReadUint16()
			item.Quantity = r.ReadUint16()
			item.Price = r.ReadUint32()
			item.Asset.Decode(l, ctx)(r, options)
			m.items = append(m.items, item)
		}
	}
}

// InteractionVisitList is the hired-merchant visit-list response
// (CEntrustedShopDlg sub_519505, v83 @0x519505, mode 0x2E): Decode2 count,
// then per entry DecodeStr name + Decode4 value (the visit count the client
// shows next to each name).
type InteractionVisitList struct {
	mode    byte
	entries []VisitListEntry
}

type VisitListEntry struct {
	Name  string
	Count uint32
}

func NewInteractionVisitList(mode byte, entries []VisitListEntry) InteractionVisitList {
	return InteractionVisitList{mode: mode, entries: entries}
}

func (m InteractionVisitList) Operation() string { return CharacterInteractionWriter }
func (m InteractionVisitList) String() string {
	return fmt.Sprintf("visit list entries [%d]", len(m.entries))
}

func (m InteractionVisitList) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(uint16(len(m.entries)))
		for _, e := range m.entries {
			w.WriteAsciiString(e.Name)
			w.WriteInt(e.Count)
		}
		return w.Bytes()
	}
}

func (m *InteractionVisitList) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadUint16())
		m.entries = make([]VisitListEntry, 0, count)
		for i := 0; i < count; i++ {
			m.entries = append(m.entries, VisitListEntry{Name: r.ReadAsciiString(), Count: r.ReadUint32()})
		}
	}
}

// InteractionBlackList is the hired-merchant blacklist-view response
// (CEntrustedShopDlg sub_5193D8, v83 @0x5193d8, mode 0x2F): Decode2 count,
// then per entry DecodeStr name.
type InteractionBlackList struct {
	mode  byte
	names []string
}

func NewInteractionBlackList(mode byte, names []string) InteractionBlackList {
	return InteractionBlackList{mode: mode, names: names}
}

func (m InteractionBlackList) Operation() string { return CharacterInteractionWriter }
func (m InteractionBlackList) String() string {
	return fmt.Sprintf("black list entries [%d]", len(m.names))
}

func (m InteractionBlackList) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(uint16(len(m.names)))
		for _, n := range m.names {
			w.WriteAsciiString(n)
		}
		return w.Bytes()
	}
}

func (m *InteractionBlackList) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadUint16())
		m.names = make([]string, 0, count)
		for i := 0; i < count; i++ {
			m.names = append(m.names, r.ReadAsciiString())
		}
	}
}
