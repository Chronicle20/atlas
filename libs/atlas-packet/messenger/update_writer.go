package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// UpdateW - mode, position, avatarBytes, name, channelId (same wire format as Add, distinguished by mode)
type UpdateW struct {
	mode        byte
	position    byte
	avatarBytes []byte
	name        string
	channelId   byte
}

func NewMessengerUpdate(mode byte, position byte, avatarBytes []byte, name string, channelId byte) UpdateW {
	return UpdateW{mode: mode, position: position, avatarBytes: avatarBytes, name: name, channelId: channelId}
}

func (m UpdateW) Mode() byte         { return m.mode }
func (m UpdateW) Position() byte     { return m.position }
func (m UpdateW) AvatarBytes() []byte { return m.avatarBytes }
func (m UpdateW) Name() string       { return m.name }
func (m UpdateW) ChannelId() byte    { return m.channelId }
func (m UpdateW) Operation() string  { return MessengerOperationWriter }

func (m UpdateW) String() string {
	return fmt.Sprintf("messenger update name [%s] position [%d]", m.name, m.position)
}

func (m UpdateW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		w.WriteByteArray(m.avatarBytes)
		w.WriteAsciiString(m.name)
		w.WriteByte(m.channelId)
		w.WriteByte(0x00)
		return w.Bytes()
	}
}

func (m *UpdateW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
