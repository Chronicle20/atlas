package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobEscortFullPathWriter = "MobEscortFullPath"

// MobEscortWaypoint is one entry of a MOB_ESCORT_FULL_PATH waypoint array.
//
// Byte layout (per loop iteration in CMob::OnEscortFullPath):
//   - x    : int32 — Decode4
//   - y    : int32 — Decode4
//   - kind : int32 — Decode4 (1 → no extra; 2 → an extra int32 follows; else
//     the client snaps y to the foothold underneath)
//   - extra : int32 — Decode4, present ONLY when kind == 2
type MobEscortWaypoint struct {
	x     int32
	y     int32
	kind  int32
	extra int32
}

func NewMobEscortWaypoint(x int32, y int32, kind int32, extra int32) MobEscortWaypoint {
	return MobEscortWaypoint{x: x, y: y, kind: kind, extra: extra}
}

func (w MobEscortWaypoint) X() int32     { return w.x }
func (w MobEscortWaypoint) Y() int32     { return w.y }
func (w MobEscortWaypoint) Kind() int32  { return w.kind }
func (w MobEscortWaypoint) Extra() int32 { return w.extra }

// MobEscortFullPath is the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath): the server hands the client the full waypoint path for
// an escort mob.
//
// Byte layout (IDA-verified):
//   - mode        : int32 — Decode4; escort mode (m_pvcActive escort field 481)
//   - count       : int32 — Decode4; number of waypoints (ZArray::_Alloc(count))
//   - waypoints   : count × {x int32, y int32, kind int32, [kind==2: extra int32]}
//   - tail        : int32 — Decode4; m_pvcActive escort field 486
//   - hasArrive   : bool  — Decode1; whether an arrive-delay follows  } arriveDelay
//   - arriveDelay : int32 — Decode4 (+ get_update_time)               } when hasArrive
//   - hasReset    : bool  — Decode1; reset flag (resets escort start when set)
//
// IDA basis: CMob::OnEscortFullPath — v95 @0x643d90, jms @0x6efa01. The harvest
// summary "8×Decode4 + Decode1 + Decode4 + Decode1" corresponds to the example of a
// 2-waypoint path (mode + count + 2×(x,y,kind) = 8 Decode4, then tail Decode4,
// arrive Decode1+Decode4, reset Decode1). v95/jms only — escort family absent in
// v83/v84/v87.
//
// packet-audit:fname CMob::OnEscortFullPath
type MobEscortFullPath struct {
	mode        int32
	waypoints   []MobEscortWaypoint
	tail        int32
	hasArrive   bool
	arriveDelay int32
	hasReset    bool
}

func NewMobEscortFullPath(mode int32, waypoints []MobEscortWaypoint, tail int32, hasArrive bool, arriveDelay int32, hasReset bool) MobEscortFullPath {
	return MobEscortFullPath{mode: mode, waypoints: waypoints, tail: tail, hasArrive: hasArrive, arriveDelay: arriveDelay, hasReset: hasReset}
}

func (m MobEscortFullPath) Mode() int32                  { return m.mode }
func (m MobEscortFullPath) Waypoints() []MobEscortWaypoint { return m.waypoints }
func (m MobEscortFullPath) Tail() int32                  { return m.tail }
func (m MobEscortFullPath) HasArrive() bool              { return m.hasArrive }
func (m MobEscortFullPath) ArriveDelay() int32           { return m.arriveDelay }
func (m MobEscortFullPath) HasReset() bool               { return m.hasReset }
func (m MobEscortFullPath) Operation() string            { return MobEscortFullPathWriter }
func (m MobEscortFullPath) String() string {
	return fmt.Sprintf("mode [%d], waypoints [%d], tail [%d], hasArrive [%t], arriveDelay [%d], hasReset [%t]",
		m.mode, len(m.waypoints), m.tail, m.hasArrive, m.arriveDelay, m.hasReset)
}

func (m MobEscortFullPath) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.mode)
		w.WriteInt32(int32(len(m.waypoints)))
		for _, wp := range m.waypoints {
			w.WriteInt32(wp.x)
			w.WriteInt32(wp.y)
			w.WriteInt32(wp.kind)
			if wp.kind == 2 {
				w.WriteInt32(wp.extra)
			}
		}
		w.WriteInt32(m.tail)
		w.WriteBool(m.hasArrive)
		if m.hasArrive {
			w.WriteInt32(m.arriveDelay)
		}
		w.WriteBool(m.hasReset)
		return w.Bytes()
	}
}

func (m *MobEscortFullPath) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadInt32()
		count := r.ReadInt32()
		m.waypoints = make([]MobEscortWaypoint, 0, count)
		for i := int32(0); i < count; i++ {
			var wp MobEscortWaypoint
			wp.x = r.ReadInt32()
			wp.y = r.ReadInt32()
			wp.kind = r.ReadInt32()
			if wp.kind == 2 {
				wp.extra = r.ReadInt32()
			}
			m.waypoints = append(m.waypoints, wp)
		}
		m.tail = r.ReadInt32()
		m.hasArrive = r.ReadBool()
		if m.hasArrive {
			m.arriveDelay = r.ReadInt32()
		}
		m.hasReset = r.ReadBool()
	}
}
