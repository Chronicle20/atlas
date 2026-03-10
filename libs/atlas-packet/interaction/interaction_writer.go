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

// InteractionEnter - visitor entering a room (pre-encoded visitor bytes)
type InteractionEnter struct {
	mode         byte
	visitorBytes []byte
}

func NewInteractionEnter(mode byte, visitorBytes []byte) InteractionEnter {
	return InteractionEnter{mode: mode, visitorBytes: visitorBytes}
}

func (m InteractionEnter) Operation() string { return CharacterInteractionWriter }
func (m InteractionEnter) String() string    { return "enter" }

func (m InteractionEnter) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.visitorBytes)
		return w.Bytes()
	}
}

func (m *InteractionEnter) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: variable-length visitor encoding
	}
}

// InteractionEnterResultSuccess - successful room entry (pre-encoded room bytes)
type InteractionEnterResultSuccess struct {
	mode      byte
	roomBytes []byte
}

func NewInteractionEnterResultSuccess(mode byte, roomBytes []byte) InteractionEnterResultSuccess {
	return InteractionEnterResultSuccess{mode: mode, roomBytes: roomBytes}
}

func (m InteractionEnterResultSuccess) Operation() string { return CharacterInteractionWriter }
func (m InteractionEnterResultSuccess) String() string    { return "enter result success" }

func (m InteractionEnterResultSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.roomBytes)
		return w.Bytes()
	}
}

func (m *InteractionEnterResultSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: variable-length room encoding
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
