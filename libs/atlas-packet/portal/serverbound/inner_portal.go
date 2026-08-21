package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const InnerPortalHandle = "InnerPortalHandle"

// InnerPortal - the client's in-map ("inner") portal teleport registration.
// The client performs the move locally and reports it; fields 3/4 are where
// the character stood BEFORE the teleport and fields 5/6 are what the
// CLIENT's own WZ says the destination portal's position is. Neither is
// adopted as authoritative — see services/atlas-channel/.../portal.EnterInner.
// packet-audit:fname CUserLocal::TryRegisterTeleport
type InnerPortal struct {
	fieldKey   byte
	portalName string
	x          int16
	y          int16
	targetX    int16
	targetY    int16
}

func (m InnerPortal) FieldKey() byte {
	return m.fieldKey
}

func (m InnerPortal) PortalName() string {
	return m.portalName
}

func (m InnerPortal) X() int16 {
	return m.x
}

func (m InnerPortal) Y() int16 {
	return m.y
}

func (m InnerPortal) TargetX() int16 {
	return m.targetX
}

func (m InnerPortal) TargetY() int16 {
	return m.targetY
}

func (m InnerPortal) Operation() string {
	return InnerPortalHandle
}

func (m InnerPortal) String() string {
	return fmt.Sprintf("fieldKey [%d], portalName [%s], x [%d], y [%d], targetX [%d], targetY [%d]", m.fieldKey, m.portalName, m.x, m.y, m.targetX, m.targetY)
}

// encodesFieldKey reports whether this version puts fieldKey on the wire.
// gms_v48's send site (0x6a5462) has no Encode1 call at all; gms_v61's send
// site (0x7aa1e3) opens with Encode1(fieldKey) immediately after the
// COutPacket constructor, followed by EncodeStr. Both sides of the boundary
// are directly decompiled — see
// docs/tasks/task-250-inner-portal-registration/structures/gms_v61.md §Gate
// decision. gms_v48 is the only in-scope version below the boundary; every
// other in-scope version (61, 72, 79, 83, 84, 87, 92, 95, jms_v185) carries
// fieldKey.
func encodesFieldKey(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS"
}

func (m InnerPortal) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if encodesFieldKey(t) {
			w.WriteByte(m.fieldKey)
		}
		w.WriteAsciiString(m.portalName)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt16(m.targetX)
		w.WriteInt16(m.targetY)
		return w.Bytes()
	}
}

func (m *InnerPortal) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if encodesFieldKey(t) {
			m.fieldKey = r.ReadByte()
		}
		m.portalName = r.ReadAsciiString()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.targetX = r.ReadInt16()
		m.targetY = r.ReadInt16()
	}
}
