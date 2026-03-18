package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Update - mode, position, avatar, name, channelId (same wire format as Add, distinguished by mode)
type Update struct {
	mode      byte
	position  byte
	avatar    model.Avatar
	name      string
	channelId byte
}

func NewMessengerUpdate(mode byte, position byte, avatar model.Avatar, name string, channelId byte) Update {
	return Update{mode: mode, position: position, avatar: avatar, name: name, channelId: channelId}
}

func (m Update) Mode() byte         { return m.mode }
func (m Update) Position() byte     { return m.position }
func (m Update) Avatar() model.Avatar { return m.avatar }
func (m Update) Name() string       { return m.name }
func (m Update) ChannelId() byte    { return m.channelId }
func (m Update) Operation() string  { return MessengerOperationWriter }

func (m Update) String() string {
	return fmt.Sprintf("messenger update name [%s] position [%d]", m.name, m.position)
}

func (m Update) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m *Update) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
		m.avatar.Decode(l, ctx)(r, options)
		m.name = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		_ = r.ReadByte() // padding
	}
}
