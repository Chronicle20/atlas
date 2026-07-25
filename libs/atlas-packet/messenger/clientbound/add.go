package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// legacyAdd reports whether this tenant uses the ancient (GMS<=28) messenger Add
// wire, which omits the trailing channelId + pad byte.
//
// The channelId + pad ARE present from GMS v48 onward: IDA-verified
// CUIMessenger::OnPacket#Add (case 0 = OnEnter) reads mode + position + avatar +
// name + channelId + pad in gms_v48 (sub_61B860 @0x61b860), gms_v61 (sub_6D144E
// @0x6d144e, the real dispatcher case-0), gms_v72 (0x777b25), gms_v79, and
// gms_v83 (0x8511fc). A prior revision gated channelId+pad off for the whole
// GMS<72 range based on sub_5BF5AE — but that is a CMiniRoomBaseDlg arm
// (dispatched by sub_5BEC69), NOT the messenger dispatcher, so the "legacy" wire
// it modelled never applied to v48/v61. Only GMS<=28 (no IDB available to verify)
// retains the pre-existing channelId-less assumption.
func legacyAdd(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.IsRegion("GMS") && t.MajorVersion() <= 28
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
