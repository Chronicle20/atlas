package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetAvatarMegaphoneWriter = "SetAvatarMegaphone"
const ClearAvatarMegaphoneWriter = "ClearAvatarMegaphone"
const AvatarMegaphoneResultWriter = "AvatarMegaphoneResult"

// SetAvatarMegaphone arms the avatar (Mega Phone / character-look) megaphone
// UI: itemId, sender name, 4 message lines, channel, whisper flag, and the
// sender's AvatarLook (design §1.2, IDA v83≡v95).
type SetAvatarMegaphone struct {
	itemId     uint32
	name       string
	lines      [4]string
	channelId  uint32
	whispersOn bool
	look       model.Avatar
}

func NewSetAvatarMegaphone(itemId uint32, name string, lines [4]string, channelId uint32, whispersOn bool, look model.Avatar) SetAvatarMegaphone {
	return SetAvatarMegaphone{
		itemId:     itemId,
		name:       name,
		lines:      lines,
		channelId:  channelId,
		whispersOn: whispersOn,
		look:       look,
	}
}

func (m SetAvatarMegaphone) ItemId() uint32     { return m.itemId }
func (m SetAvatarMegaphone) Name() string       { return m.name }
func (m SetAvatarMegaphone) Lines() [4]string   { return m.lines }
func (m SetAvatarMegaphone) ChannelId() uint32  { return m.channelId }
func (m SetAvatarMegaphone) WhispersOn() bool   { return m.whispersOn }
func (m SetAvatarMegaphone) Look() model.Avatar { return m.look }
func (m SetAvatarMegaphone) Operation() string  { return SetAvatarMegaphoneWriter }
func (m SetAvatarMegaphone) String() string {
	return fmt.Sprintf("set avatar megaphone itemId [%d] name [%s]", m.itemId, m.name)
}

func (m SetAvatarMegaphone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteAsciiString(m.name)
		for _, line := range m.lines {
			w.WriteAsciiString(line)
		}
		w.WriteInt(m.channelId)
		w.WriteBool(m.whispersOn)
		w.WriteByteArray(m.look.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *SetAvatarMegaphone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = r.ReadUint32()
		m.name = r.ReadAsciiString()
		for i := range m.lines {
			m.lines[i] = r.ReadAsciiString()
		}
		m.channelId = r.ReadUint32()
		m.whispersOn = r.ReadBool()
		look := model.Avatar{}
		look.Decode(l, ctx)(r, options)
		m.look = look
	}
}

// ClearAvatarMegaphone tears down the avatar megaphone UI. Cosmic sends a
// single guard byte (1); the client's clear handler is idempotent regardless
// of the byte value.
type ClearAvatarMegaphone struct {
	flag byte
}

func NewClearAvatarMegaphone() ClearAvatarMegaphone {
	return ClearAvatarMegaphone{flag: 1}
}

func (m ClearAvatarMegaphone) Flag() byte        { return m.flag }
func (m ClearAvatarMegaphone) Operation() string { return ClearAvatarMegaphoneWriter }
func (m ClearAvatarMegaphone) String() string    { return "clear avatar megaphone" }

func (m ClearAvatarMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(1)
		return w.Bytes()
	}
}

func (m *ClearAvatarMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.flag = r.ReadByte()
	}
}

// AvatarMegaphoneResult carries a config-resolved result byte and, when the
// selected reason semantically carries a trailing notice, the message text.
// hasMessage is set explicitly by the constructor rather than derived from
// the resolved byte value (A1.3): comparing a resolved byte against literals
// 83/84 would silently break on any tenant whose errorCodes table maps those
// reasons to different bytes. NewAvatarMegaphoneResult(code, "") always
// writes code-only, regardless of what byte the tenant resolved.
type AvatarMegaphoneResult struct {
	code       byte
	hasMessage bool
	message    string
}

func NewAvatarMegaphoneResult(code byte, message string) AvatarMegaphoneResult {
	return AvatarMegaphoneResult{code: code, hasMessage: message != "", message: message}
}

func (m AvatarMegaphoneResult) Code() byte        { return m.code }
func (m AvatarMegaphoneResult) HasMessage() bool  { return m.hasMessage }
func (m AvatarMegaphoneResult) Message() string   { return m.message }
func (m AvatarMegaphoneResult) Operation() string { return AvatarMegaphoneResultWriter }
func (m AvatarMegaphoneResult) String() string {
	return fmt.Sprintf("avatar megaphone result code [%d]", m.code)
}

func (m AvatarMegaphoneResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		if m.hasMessage {
			w.WriteAsciiString(m.message)
		}
		return w.Bytes()
	}
}

func (m *AvatarMegaphoneResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		if r.Available() > 0 {
			m.hasMessage = true
			m.message = r.ReadAsciiString()
		}
	}
}
