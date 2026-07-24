package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const MapChangeHandle = "MapChangeHandle"

// Change - CField::SendTransferFieldPacket
// packet-audit:fname CField::SendTransferFieldRequest
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
		// v95 client (CField::SendTransferFieldRequest @0x5345c0) emits the
		// target x/y only when a portal name is supplied (sPortal != NULL); the
		// null-portal Revive path sends an empty name and no coordinates.
		if len(m.portalName) != 0 {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		w.WriteByte(m.unused)
		w.WriteByte(m.premium)
		// The chase flag (s_bChase / dword_975DD4 in v61, dword_AA4E60 in v72,
		// dword_B0D450 in v79, dword_80D3EC in v48) is emitted UNCONDITIONALLY by
		// every GMS client. v61 IDA: CField::SendTransferFieldRequest @0x4e8f58
		// Encode1(dword_975DD4) (unconditional), then Encode4(targetX)/Encode4(targetY)
		// when set. v72 IDA: @0x5148b1 Encode1(dword_AA4E60) @0x51499f — structurally
		// identical to the v79 @0x51ba3e pattern. v48 IDA: @0x4c5733
		// Encode1(dword_80D3EC) @0x4c5822 (unconditional), then Encode4(a5)/Encode4(a6)
		// under `if (dword_80D3EC)`. The legacy gate was >=79 (wrongly dropped the byte
		// for v72), then >=72 (wrongly dropped it for v61), then >=61 (wrongly dropped it
		// for v48); lowered to >=48 (oldest GMS). v61/72/79/83/84/87/95 unchanged.
		if t.Region() == "GMS" && t.MajorVersion() >= 48 {
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
		// v95 client encodes x/y only alongside a non-empty portal name; see
		// CField::SendTransferFieldRequest @0x5345c0 (test ebx,ebx / jz @0x5346d9).
		if len(m.portalName) != 0 {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		}
		m.unused = r.ReadByte()
		m.premium = r.ReadByte()
		// chase flag present for all GMS incl. v48 (see Encode comment): >=48 legacy gate.
		if t.Region() == "GMS" && t.MajorVersion() >= 48 {
			m.chase = r.ReadBool()
		}
		if m.chase {
			m.targetX = r.ReadInt32()
			m.targetY = r.ReadInt32()
		}
	}
}
