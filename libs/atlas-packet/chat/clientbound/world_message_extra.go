package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// WorldMessageUnknown3 covers Unknown3 and Unknown4 modes - mode, message, "doo" string, operator uint32
type WorldMessageUnknown3 struct {
	mode     byte
	message  string
	operator uint32
}

func NewWorldMessageUnknown3(mode byte, message string, operator uint32) WorldMessageUnknown3 {
	return WorldMessageUnknown3{mode: mode, message: message, operator: operator}
}

func NewWorldMessageUnknown4(mode byte, message string, operator uint32) WorldMessageUnknown3 {
	return WorldMessageUnknown3{mode: mode, message: message, operator: operator}
}

func (m WorldMessageUnknown3) Mode() byte      { return m.mode }
func (m WorldMessageUnknown3) Message() string { return m.message }
func (m WorldMessageUnknown3) Operator() uint32 { return m.operator }

func (m WorldMessageUnknown3) Operation() string { return WorldMessageWriter }
func (m WorldMessageUnknown3) String() string {
	return fmt.Sprintf("world message unknown mode [%d]", m.mode)
}

func (m WorldMessageUnknown3) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteAsciiString("doo")
		w.WriteInt(m.operator)
		return w.Bytes()
	}
}

func (m *WorldMessageUnknown3) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		_ = r.ReadAsciiString()
		m.operator = r.ReadUint32()
	}
}

// WorldMessageUnknown7 - mode, message, int(0)
type WorldMessageUnknown7 struct {
	mode    byte
	message string
}

func NewWorldMessageUnknown7(mode byte, message string) WorldMessageUnknown7 {
	return WorldMessageUnknown7{mode: mode, message: message}
}

func (m WorldMessageUnknown7) Mode() byte      { return m.mode }
func (m WorldMessageUnknown7) Message() string { return m.message }

func (m WorldMessageUnknown7) Operation() string { return WorldMessageWriter }
func (m WorldMessageUnknown7) String() string    { return "world message unknown 7" }

func (m WorldMessageUnknown7) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteInt(uint32(0))
		return w.Bytes()
	}
}

func (m *WorldMessageUnknown7) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		_ = r.ReadUint32()
	}
}

// WorldMessageUnknown8 - mode, message, channelId, whispersOn
type WorldMessageUnknown8 struct {
	mode       byte
	message    string
	channelId  byte
	whispersOn bool
}

func NewWorldMessageUnknown8(mode byte, message string, channelId byte, whispersOn bool) WorldMessageUnknown8 {
	return WorldMessageUnknown8{mode: mode, message: message, channelId: channelId, whispersOn: whispersOn}
}

func (m WorldMessageUnknown8) Mode() byte       { return m.mode }
func (m WorldMessageUnknown8) Message() string  { return m.message }
func (m WorldMessageUnknown8) ChannelId() byte  { return m.channelId }
func (m WorldMessageUnknown8) WhispersOn() bool { return m.whispersOn }

func (m WorldMessageUnknown8) Operation() string { return WorldMessageWriter }
func (m WorldMessageUnknown8) String() string    { return "world message unknown 8" }

func (m WorldMessageUnknown8) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.channelId)
		w.WriteBool(m.whispersOn)
		return w.Bytes()
	}
}

func (m *WorldMessageUnknown8) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		m.whispersOn = r.ReadBool()
	}
}

// WorldMessageWeather - mode, message, weatherItemId
type WorldMessageWeather struct {
	mode          byte
	message       string
	weatherItemId uint32
}

func NewWorldMessageWeather(mode byte, message string, weatherItemId uint32) WorldMessageWeather {
	return WorldMessageWeather{mode: mode, message: message, weatherItemId: weatherItemId}
}

func (m WorldMessageWeather) Mode() byte          { return m.mode }
func (m WorldMessageWeather) Message() string     { return m.message }
func (m WorldMessageWeather) WeatherItemId() uint32 { return m.weatherItemId }

func (m WorldMessageWeather) Operation() string { return WorldMessageWriter }
func (m WorldMessageWeather) String() string {
	return fmt.Sprintf("world message weather itemId [%d]", m.weatherItemId)
}

func (m WorldMessageWeather) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteInt(m.weatherItemId)
		return w.Bytes()
	}
}

func (m *WorldMessageWeather) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.weatherItemId = r.ReadUint32()
	}
}
