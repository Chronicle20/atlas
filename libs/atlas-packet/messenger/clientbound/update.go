package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Update - mode, position, avatar
// IDA: CUIMessenger::OnPacket mode=7 → OnAvatar: Decode1(position) + AvatarLook::Decode only.
// The client does not read name or channelId for avatar-update packets.
type Update struct {
	mode     byte
	position byte
	avatar   model.Avatar
}

func NewMessengerUpdate(mode byte, position byte, avatar model.Avatar) Update {
	return Update{mode: mode, position: position, avatar: avatar}
}

func (m Update) Mode() byte           { return m.mode }
func (m Update) Position() byte       { return m.position }
func (m Update) Avatar() model.Avatar { return m.avatar }
func (m Update) Operation() string    { return MessengerOperationWriter }

func (m Update) String() string {
	return fmt.Sprintf("messenger update position [%d]", m.position)
}

func (m Update) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Update) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
		m.avatar.Decode(l, ctx)(r, options)
	}
}
