package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Add - mode, position, avatar, name, channelId
// packet-audit:fname CUIMessenger::OnPacket#Add
type Add struct {
	mode      byte
	position  byte
	avatar    model.Avatar
	name      string
	channelId byte
}

func NewMessengerAdd(mode byte, position byte, avatar model.Avatar, name string, channelId byte) Add {
	return Add{mode: mode, position: position, avatar: avatar, name: name, channelId: channelId}
}

func (m Add) Mode() byte           { return m.mode }
func (m Add) Position() byte       { return m.position }
func (m Add) Avatar() model.Avatar { return m.avatar }
func (m Add) Name() string         { return m.name }
func (m Add) ChannelId() byte      { return m.channelId }
func (m Add) Operation() string    { return MessengerOperationWriter }

func (m Add) String() string {
	return fmt.Sprintf("messenger add name [%s] position [%d]", m.name, m.position)
}

// legacyAdd reports whether this tenant uses the pre-v72 messenger Add wire.
// v61 CUIMessenger::OnPacket#Add (sub_5BF5AE @0x5bf5ae, GMS_v61.1_U_DEVM.exe)
// reads Decode1(position) + avatar + DecodeStr(name) ONLY — no channelId and no
// trailing pad byte. The channelId + pad were added in GMS>=72 (v72 OnEnter),
// so the legacy range (GMS <72) omits them. Avatar encoding is unchanged across
// this boundary (model.Avatar gates only on GMS<=28 vs >28; v61 is >28 == v83).
func legacyAdd(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.IsRegion("GMS") && !t.MajorAtLeast(72)
}

func (m Add) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	legacy := legacyAdd(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
		w.WriteAsciiString(m.name)
		if !legacy {
			w.WriteByte(m.channelId)
			w.WriteByte(0x00)
		}
		return w.Bytes()
	}
}

func (m *Add) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	legacy := legacyAdd(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
		m.avatar.Decode(l, ctx)(r, options)
		m.name = r.ReadAsciiString()
		if !legacy {
			m.channelId = r.ReadByte()
			_ = r.ReadByte() // padding
		}
	}
}
