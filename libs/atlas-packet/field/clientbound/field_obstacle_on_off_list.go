package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const FieldObstacleOnOffListWriter = "FieldObstacleOnOffList"

type ObstacleState struct {
	name  string
	state uint32
}

func NewObstacleState(name string, state uint32) ObstacleState {
	return ObstacleState{name: name, state: state}
}

func (o ObstacleState) Name() string  { return o.name }
func (o ObstacleState) State() uint32 { return o.state }

// packet-audit:fname CField::OnFieldObstacleOnOffStatus
//
// Version divergence: v83+ (GMS>=83, JMS) send a count-prefixed LIST of
// {name, state} entries. The oldest GMS client (v48) instead sends a SINGLE
// obstacle: IDA CField::OnFieldObstacleOnOffStatus = sub_4C930A @0x4c930a reads
// Decode1(flag) @0x4c9328 + Decode4(itemId) @0x4c932e, then — only when itemId!=0
// (the GetItemInfo block) and flag==0 — DecodeStr(name) @0x4c9558. The legacy
// single-obstacle shape is carried in the legacyFlag/legacyItemId/legacyName
// fields under a GMS<61 gate; the v61+ list path is unchanged.
// packet-audit:fname CField::OnFieldObstacleOnOffStatus
type FieldObstacleOnOffList struct {
	obstacles []ObstacleState
	// Legacy (GMS<61 / v48) single-obstacle fields — see type doc.
	legacyFlag   byte
	legacyItemId uint32
	legacyName   string
}

func NewFieldObstacleOnOffList(obstacles []ObstacleState) FieldObstacleOnOffList {
	return FieldObstacleOnOffList{obstacles: obstacles}
}

// NewFieldObstacleLegacy builds the GMS<61 single-obstacle variant (v48). The
// name is only on the wire when itemId!=0 and flag==0 (see the sub_4C930A decode).
func NewFieldObstacleLegacy(flag byte, itemId uint32, name string) FieldObstacleOnOffList {
	return FieldObstacleOnOffList{legacyFlag: flag, legacyItemId: itemId, legacyName: name}
}

func (m FieldObstacleOnOffList) Obstacles() []ObstacleState { return m.obstacles }
func (m FieldObstacleOnOffList) LegacyFlag() byte           { return m.legacyFlag }
func (m FieldObstacleOnOffList) LegacyItemId() uint32       { return m.legacyItemId }
func (m FieldObstacleOnOffList) LegacyName() string         { return m.legacyName }

func (m FieldObstacleOnOffList) Operation() string { return FieldObstacleOnOffListWriter }
func (m FieldObstacleOnOffList) String() string {
	return fmt.Sprintf("obstacles [%d]", len(m.obstacles))
}

func (m FieldObstacleOnOffList) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() < 61 {
			// v48 single obstacle: sub_4C930A @0x4c930a.
			w.WriteByte(m.legacyFlag)  // Decode1(flag) @0x4c9328
			w.WriteInt(m.legacyItemId) // Decode4(itemId) @0x4c932e
			// name only when itemId!=0 (GetItemInfo block @0x4c9375) and flag==0
			// (DecodeStr @0x4c9558 under if(!flag)).
			if m.legacyItemId != 0 && m.legacyFlag == 0 {
				w.WriteAsciiString(m.legacyName)
			}
			return w.Bytes()
		}
		w.WriteInt(uint32(len(m.obstacles)))
		for _, o := range m.obstacles {
			w.WriteAsciiString(o.name)
			w.WriteInt(o.state)
		}
		return w.Bytes()
	}
}

func (m *FieldObstacleOnOffList) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() < 61 {
			m.legacyFlag = r.ReadByte()     // Decode1 @0x4c9328
			m.legacyItemId = r.ReadUint32() // Decode4 @0x4c932e
			if m.legacyItemId != 0 && m.legacyFlag == 0 {
				m.legacyName = r.ReadAsciiString() // DecodeStr @0x4c9558
			}
			return
		}
		count := r.ReadUint32()
		m.obstacles = make([]ObstacleState, 0, count)
		for i := uint32(0); i < count; i++ {
			name := r.ReadAsciiString()
			state := r.ReadUint32()
			m.obstacles = append(m.obstacles, ObstacleState{name: name, state: state})
		}
	}
}
