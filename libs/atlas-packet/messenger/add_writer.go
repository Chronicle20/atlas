package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// AddW - mode, position, avatar, name, channelId
type AddW struct {
	mode      byte
	position  byte
	avatar    model.Avatar
	name      string
	channelId byte
}

func NewMessengerAdd(mode byte, position byte, avatar model.Avatar, name string, channelId byte) AddW {
	return AddW{mode: mode, position: position, avatar: avatar, name: name, channelId: channelId}
}

func (m AddW) Mode() byte         { return m.mode }
func (m AddW) Position() byte     { return m.position }
func (m AddW) Avatar() model.Avatar { return m.avatar }
func (m AddW) Name() string       { return m.name }
func (m AddW) ChannelId() byte    { return m.channelId }
func (m AddW) Operation() string  { return MessengerOperationWriter }

func (m AddW) String() string {
	return fmt.Sprintf("messenger add name [%s] position [%d]", m.name, m.position)
}

func (m AddW) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
		w.WriteAsciiString(m.name)
		w.WriteByte(m.channelId)
		w.WriteByte(0x00)
		return w.Bytes()
	}
}

func (m *AddW) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
		m.avatar.Decode(l, ctx)(r, options)
		m.name = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		_ = r.ReadByte() // padding
	}
}
