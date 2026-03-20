package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WorldMessageWriter = "WorldMessage"

// WorldMessageSimple covers Notice(0), PopUp(1), Megaphone(2), PinkText(5)
type WorldMessageSimple struct {
	mode    byte
	message string
}

func NewWorldMessageSimple(mode byte, message string) WorldMessageSimple {
	return WorldMessageSimple{mode: mode, message: message}
}

func (m WorldMessageSimple) Mode() byte      { return m.mode }
func (m WorldMessageSimple) Message() string { return m.message }

func (m WorldMessageSimple) Operation() string { return WorldMessageWriter }
func (m WorldMessageSimple) String() string {
	return fmt.Sprintf("world message mode [%d]", m.mode)
}

func (m WorldMessageSimple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *WorldMessageSimple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}

// WorldMessageTopScroll - mode(4), bool(hasMessage), message
type WorldMessageTopScroll struct {
	mode    byte
	message string
}

func NewWorldMessageTopScroll(mode byte, message string) WorldMessageTopScroll {
	return WorldMessageTopScroll{mode: mode, message: message}
}

func (m WorldMessageTopScroll) Mode() byte      { return m.mode }
func (m WorldMessageTopScroll) Message() string { return m.message }

func (m WorldMessageTopScroll) Operation() string { return WorldMessageWriter }
func (m WorldMessageTopScroll) String() string    { return "world message top scroll" }

func (m WorldMessageTopScroll) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(len(m.message) > 0)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *WorldMessageTopScroll) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadBool()
		m.message = r.ReadAsciiString()
	}
}

// WorldMessageSuperMegaphone - mode(3), message, channel, whispersOn
type WorldMessageSuperMegaphone struct {
	mode       byte
	message    string
	channelId  byte
	whispersOn bool
}

func NewWorldMessageSuperMegaphone(mode byte, message string, channelId byte, whispersOn bool) WorldMessageSuperMegaphone {
	return WorldMessageSuperMegaphone{mode: mode, message: message, channelId: channelId, whispersOn: whispersOn}
}

func (m WorldMessageSuperMegaphone) Mode() byte        { return m.mode }
func (m WorldMessageSuperMegaphone) Message() string   { return m.message }
func (m WorldMessageSuperMegaphone) ChannelId() byte   { return m.channelId }
func (m WorldMessageSuperMegaphone) WhispersOn() bool  { return m.whispersOn }

func (m WorldMessageSuperMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageSuperMegaphone) String() string    { return "world message super megaphone" }

func (m WorldMessageSuperMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.channelId)
		w.WriteBool(m.whispersOn)
		return w.Bytes()
	}
}

func (m *WorldMessageSuperMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		m.whispersOn = r.ReadBool()
	}
}

// WorldMessageBlueText - mode(6), message, itemId
type WorldMessageBlueText struct {
	mode    byte
	message string
	itemId  uint32
}

func NewWorldMessageBlueText(mode byte, message string, itemId uint32) WorldMessageBlueText {
	return WorldMessageBlueText{mode: mode, message: message, itemId: itemId}
}

func (m WorldMessageBlueText) Mode() byte      { return m.mode }
func (m WorldMessageBlueText) Message() string { return m.message }
func (m WorldMessageBlueText) ItemId() uint32  { return m.itemId }

func (m WorldMessageBlueText) Operation() string { return WorldMessageWriter }
func (m WorldMessageBlueText) String() string    { return "world message blue text" }

func (m WorldMessageBlueText) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *WorldMessageBlueText) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.itemId = r.ReadUint32()
	}
}

// WorldMessageItemMegaphone - mode, message, channel, whispersOn, hasItem(true), slot
type WorldMessageItemMegaphone struct {
	mode       byte
	message    string
	channelId  byte
	whispersOn bool
	slot       int32
}

func NewWorldMessageItemMegaphone(mode byte, message string, channelId byte, whispersOn bool, slot int32) WorldMessageItemMegaphone {
	return WorldMessageItemMegaphone{mode: mode, message: message, channelId: channelId, whispersOn: whispersOn, slot: slot}
}

func (m WorldMessageItemMegaphone) Mode() byte        { return m.mode }
func (m WorldMessageItemMegaphone) Message() string   { return m.message }
func (m WorldMessageItemMegaphone) ChannelId() byte   { return m.channelId }
func (m WorldMessageItemMegaphone) WhispersOn() bool  { return m.whispersOn }
func (m WorldMessageItemMegaphone) Slot() int32       { return m.slot }

func (m WorldMessageItemMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageItemMegaphone) String() string    { return "world message item megaphone" }

func (m WorldMessageItemMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.channelId)
		w.WriteBool(m.whispersOn)
		w.WriteBool(true)
		w.WriteInt32(m.slot)
		return w.Bytes()
	}
}

func (m *WorldMessageItemMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		m.whispersOn = r.ReadBool()
		_ = r.ReadBool()
		m.slot = r.ReadInt32()
	}
}

// WorldMessageYellowMegaphone - mode, message, channel
type WorldMessageYellowMegaphone struct {
	mode      byte
	message   string
	channelId byte
}

func NewWorldMessageYellowMegaphone(mode byte, message string, channelId byte) WorldMessageYellowMegaphone {
	return WorldMessageYellowMegaphone{mode: mode, message: message, channelId: channelId}
}

func (m WorldMessageYellowMegaphone) Mode() byte      { return m.mode }
func (m WorldMessageYellowMegaphone) Message() string { return m.message }
func (m WorldMessageYellowMegaphone) ChannelId() byte { return m.channelId }

func (m WorldMessageYellowMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageYellowMegaphone) String() string    { return "world message yellow megaphone" }

func (m WorldMessageYellowMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.channelId)
		return w.Bytes()
	}
}

func (m *WorldMessageYellowMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.channelId = r.ReadByte()
	}
}

// WorldMessageMultiMegaphone - mode, messages, channel, whispersOn
type WorldMessageMultiMegaphone struct {
	mode       byte
	messages   []string
	channelId  byte
	whispersOn bool
}

func NewWorldMessageMultiMegaphone(mode byte, messages []string, channelId byte, whispersOn bool) WorldMessageMultiMegaphone {
	return WorldMessageMultiMegaphone{mode: mode, messages: messages, channelId: channelId, whispersOn: whispersOn}
}

func (m WorldMessageMultiMegaphone) Mode() byte         { return m.mode }
func (m WorldMessageMultiMegaphone) Messages() []string { return m.messages }
func (m WorldMessageMultiMegaphone) ChannelId() byte    { return m.channelId }
func (m WorldMessageMultiMegaphone) WhispersOn() bool   { return m.whispersOn }

func (m WorldMessageMultiMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageMultiMegaphone) String() string    { return "world message multi megaphone" }

func (m WorldMessageMultiMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.messages[0])
		w.WriteByte(byte(len(m.messages)))
		for _, msg := range m.messages[1:] {
			w.WriteAsciiString(msg)
		}
		w.WriteByte(m.channelId)
		w.WriteBool(m.whispersOn)
		return w.Bytes()
	}
}

func (m *WorldMessageMultiMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		firstMsg := r.ReadAsciiString()
		count := r.ReadByte()
		m.messages = make([]string, 1, count)
		m.messages[0] = firstMsg
		for i := byte(1); i < count; i++ {
			m.messages = append(m.messages, r.ReadAsciiString())
		}
		m.channelId = r.ReadByte()
		m.whispersOn = r.ReadBool()
	}
}

// WorldMessageGachapon - mode, message (character name), unk uint32, townName, itemId
type WorldMessageGachapon struct {
	mode     byte
	message  string
	townName string
	itemId   uint32
}

func NewWorldMessageGachapon(mode byte, message string, townName string, itemId uint32) WorldMessageGachapon {
	return WorldMessageGachapon{mode: mode, message: message, townName: townName, itemId: itemId}
}

func (m WorldMessageGachapon) Mode() byte       { return m.mode }
func (m WorldMessageGachapon) Message() string  { return m.message }
func (m WorldMessageGachapon) TownName() string { return m.townName }
func (m WorldMessageGachapon) ItemId() uint32   { return m.itemId }

func (m WorldMessageGachapon) Operation() string { return WorldMessageWriter }
func (m WorldMessageGachapon) String() string    { return "world message gachapon" }

func (m WorldMessageGachapon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteInt(0)
		w.WriteAsciiString(m.townName)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *WorldMessageGachapon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		_ = r.ReadUint32()
		m.townName = r.ReadAsciiString()
		m.itemId = r.ReadUint32()
	}
}
