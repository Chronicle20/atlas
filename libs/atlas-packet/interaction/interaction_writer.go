package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterInteractionWriter = "CharacterInteraction"

// InteractionInvite - invite to a mini room
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
type InteractionEnter struct {
	mode    byte
	visitor Visitor
}

func NewInteractionEnter(mode byte, visitor Visitor) InteractionEnter {
	return InteractionEnter{mode: mode, visitor: visitor}
}

func (m InteractionEnter) Operation() string { return CharacterInteractionWriter }
func (m InteractionEnter) String() string    { return "enter" }
func (m InteractionEnter) Visitor() Visitor  { return m.visitor }

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
type InteractionEnterResultSuccess struct {
	mode byte
	room Room
}

func NewInteractionEnterResultSuccess(mode byte, room Room) InteractionEnterResultSuccess {
	return InteractionEnterResultSuccess{mode: mode, room: room}
}

func (m InteractionEnterResultSuccess) Operation() string { return CharacterInteractionWriter }
func (m InteractionEnterResultSuccess) String() string    { return "enter result success" }
func (m InteractionEnterResultSuccess) Room() Room        { return m.room }

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
