package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MapChangeHandle = "MapChangeHandle"

// Change - CField::SendTransferFieldPacket
type Change struct {
	cashShopReturn bool
	fieldKey       byte
	targetId       uint32
	portalName     string
	x              int16
	y              int16
	unused         byte
	premium        byte
	chase          bool
	targetX        int32
	targetY        int32
}

func (m Change) CashShopReturn() bool { return m.cashShopReturn }
func (m Change) FieldKey() byte       { return m.fieldKey }
func (m Change) TargetId() uint32     { return m.targetId }
func (m Change) PortalName() string   { return m.portalName }
func (m Change) X() int16             { return m.x }
func (m Change) Y() int16             { return m.y }
func (m Change) Premium() byte        { return m.premium }
func (m Change) Chase() bool          { return m.chase }
func (m Change) TargetX() int32       { return m.targetX }
func (m Change) TargetY() int32       { return m.targetY }

func (m Change) Operation() string {
	return MapChangeHandle
}

func (m Change) String() string {
	if m.cashShopReturn {
		return "cashShopReturn [true]"
	}
	return fmt.Sprintf("fieldKey [%d], targetId [%d], portalName [%s], x [%d], y [%d], premium [%d], chase [%t]", m.fieldKey, m.targetId, m.portalName, m.x, m.y, m.premium, m.chase)
}

func (m Change) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if m.cashShopReturn {
			return w.Bytes()
		}
		w.WriteByte(m.fieldKey)
		w.WriteInt(m.targetId)
		w.WriteAsciiString(m.portalName)
		if len(m.portalName) == 0 {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		w.WriteByte(m.unused)
		w.WriteByte(m.premium)
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			w.WriteBool(m.chase)
		}
		if m.chase {
			w.WriteInt32(m.targetX)
			w.WriteInt32(m.targetY)
		}
		return w.Bytes()
	}
}

func (m *Change) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if r.Available() == 0 {
			m.cashShopReturn = true
			return
		}
		m.fieldKey = r.ReadByte()
		m.targetId = r.ReadUint32()
		m.portalName = r.ReadAsciiString()
		if len(m.portalName) == 0 {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		}
		m.unused = r.ReadByte()
		m.premium = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			m.chase = r.ReadBool()
		}
		if m.chase {
			m.targetX = r.ReadInt32()
			m.targetY = r.ReadInt32()
		}
	}
}
