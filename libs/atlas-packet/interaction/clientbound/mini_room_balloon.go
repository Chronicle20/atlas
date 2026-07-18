package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// MiniRoomBalloon - UPDATE_CHAR_BOX game/shop balloon shown over the mini
// room owner's head. This is NOT a dispatcher-family arm: there is no mode
// byte and no WithResolvedCode resolution — the writer name alone
// (interaction.MiniRoomWriter) identifies the opcode.
//
// ida-notes.md §G3 (docs/tasks/task-133-miniroom-minigames/ida-notes.md)
// confirms the read order is identical on gms_v83 and gms_v95 and uniform
// across every MiniRoomType (no per-roomType branch) — CUser::OnMiniRoomBalloon
// reads roomType then, when roomType != 0, the trailing fields below. The
// leading int32 characterId on the wire is consumed by the dispatcher
// CUserPool::OnUserCommonPacket (v83 @ 0x972401) to resolve the target CUser
// before routing opcode 165 (0xA5) to the handler; it is not read by
// OnMiniRoomBalloon itself, but it is still part of the wire body this writer
// produces.
//
// This wire shape exactly matches the legacy MiniRoomBase.Spawn writer
// (mini_room.go:69-85), which already encodes it for shop room types
// (personal/merchant). This struct is the audited equivalent used for the
// game (Omok/Match Cards) balloon path; both ultimately serialize the same
// bytes for CUser::OnMiniRoomBalloon.
// packet-audit:fname CUser::OnMiniRoomBalloon
type MiniRoomBalloon struct {
	characterId uint32
	roomType    byte
	roomId      uint32
	title       string
	hasPassword bool
	pieceType   byte
	occupancy   byte
	capacity    byte
	inProgress  bool
}

func NewMiniRoomBalloon(characterId uint32, roomType byte, roomId uint32, title string, hasPassword bool, pieceType byte, occupancy byte, capacity byte, inProgress bool) MiniRoomBalloon {
	return MiniRoomBalloon{
		characterId: characterId,
		roomType:    roomType,
		roomId:      roomId,
		title:       title,
		hasPassword: hasPassword,
		pieceType:   pieceType,
		occupancy:   occupancy,
		capacity:    capacity,
		inProgress:  inProgress,
	}
}

func (m MiniRoomBalloon) Operation() string { return interaction.MiniRoomWriter }
func (m MiniRoomBalloon) String() string {
	return fmt.Sprintf("mini room balloon characterId [%d] roomType [%d] roomId [%d] title [%s]", m.characterId, m.roomType, m.roomId, m.title)
}

func (m MiniRoomBalloon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.roomType)
		w.WriteInt(m.roomId)
		w.WriteAsciiString(m.title)
		w.WriteBool(m.hasPassword)
		w.WriteByte(m.pieceType)
		w.WriteByte(m.occupancy)
		w.WriteByte(m.capacity)
		w.WriteBool(m.inProgress)
		return w.Bytes()
	}
}

func (m *MiniRoomBalloon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.roomType = r.ReadByte()
		m.roomId = r.ReadUint32()
		m.title = r.ReadAsciiString()
		m.hasPassword = r.ReadBool()
		m.pieceType = r.ReadByte()
		m.occupancy = r.ReadByte()
		m.capacity = r.ReadByte()
		m.inProgress = r.ReadBool()
	}
}

// MiniRoomBalloonRemove - UPDATE_CHAR_BOX with roomType == 0, which
// CUser::OnMiniRoomBalloon treats as "destroy the balloon"
// (CChatBalloon::DestroyMiniRoomBalloon) and reads no trailing fields for.
// Same handler as MiniRoomBalloon; see ida-notes.md §G3 for the shared read
// order.
// packet-audit:fname CUser::OnMiniRoomBalloon#Remove
type MiniRoomBalloonRemove struct {
	characterId uint32
}

func NewMiniRoomBalloonRemove(characterId uint32) MiniRoomBalloonRemove {
	return MiniRoomBalloonRemove{characterId: characterId}
}

func (m MiniRoomBalloonRemove) Operation() string { return interaction.MiniRoomWriter }
func (m MiniRoomBalloonRemove) String() string {
	return fmt.Sprintf("mini room balloon remove characterId [%d]", m.characterId)
}

func (m MiniRoomBalloonRemove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *MiniRoomBalloonRemove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		_ = r.ReadByte() // roomType, always 0 for the remove shape
	}
}

// MiniRoomBalloonBody and MiniRoomBalloonRemoveBody are the plain body funcs
// channel Tasks 18/19 consume verbatim. There is no mode byte to resolve —
// this packet is not a dispatcher-family arm.
func MiniRoomBalloonBody(characterId uint32, roomType byte, roomId uint32, title string, hasPassword bool, pieceType byte, occupancy byte, capacity byte, inProgress bool) func(logrus.FieldLogger, context.Context) func(options map[string]interface{}) []byte {
	return NewMiniRoomBalloon(characterId, roomType, roomId, title, hasPassword, pieceType, occupancy, capacity, inProgress).Encode
}

func MiniRoomBalloonRemoveBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(options map[string]interface{}) []byte {
	return NewMiniRoomBalloonRemove(characterId).Encode
}
