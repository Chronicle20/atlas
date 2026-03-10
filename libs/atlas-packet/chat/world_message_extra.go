package chat

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

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
