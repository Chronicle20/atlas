package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobEscortFullPathWriter = "MobEscortFullPath"

// MobEscortWaypoint is one entry of the MOB_ESCORT_FULL_PATH waypoint array —
// a CVecCtrlMob::EscortDest (20 bytes) in the client.
//
// Byte layout (per loop iteration in CMob::OnEscortFullPath):
//   - x            : int32 — Decode4; EscortDest::dp.x
//   - y            : int32 — Decode4; EscortDest::dp.y
//   - attr         : int32 — Decode4; EscortDest::nAttr. attr==1 zeroes the entry's
//     ZMass; otherwise the client fills it from
//     CWvsPhysicalSpace2D::GetFootholdUnderneath(x, y).
//   - stopDuration : int32 — Decode4; EscortDest::nStopDuration. Present ONLY when
//     attr == 2.
type MobEscortWaypoint struct {
	x            int32
	y            int32
	attr         int32
	stopDuration int32
}

func NewMobEscortWaypoint(x int32, y int32, attr int32, stopDuration int32) MobEscortWaypoint {
	return MobEscortWaypoint{x: x, y: y, attr: attr, stopDuration: stopDuration}
}

func (w MobEscortWaypoint) X() int32            { return w.x }
func (w MobEscortWaypoint) Y() int32            { return w.y }
func (w MobEscortWaypoint) Attr() int32         { return w.attr }
func (w MobEscortWaypoint) StopDuration() int32 { return w.stopDuration }

// MobEscortFullPath is the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath): the server hands the client the full waypoint path for
// an escort mob. The handler writes into the mob's CVecCtrlMob (1960 bytes).
//
// Byte layout (IDA-derived; identical in all three routed versions):
//   - count            : int32 — Decode4; waypoint count, the
//     ZArray<CVecCtrlMob::EscortDest>::_Alloc bound. Derived from len(waypoints);
//     not a struct field.
//   - oldDestX         : int32 — Decode4; CVecCtrlMob::m_Old_Dest.dp.x
//   - oldDestY         : int32 — Decode4; CVecCtrlMob::m_Old_Dest.dp.y
//     (m_Old_Dest.nAttr is zeroed and m_Old_Dest.ZMass comes from
//     CMob::GetZMass(); neither is on the wire.)
//   - waypoints        : count × {x, y, attr int32, [attr==2: stopDuration int32]}
//   - currentDestIndex : int32 — Decode4; CVecCtrlMob::m_nCurrentDestIndex, the
//     index into the waypoint array the escort tick reads.
//   - hasStopDuration  : bool  — Decode1                       } stopDuration
//   - stopDuration     : int32 — Decode4, only when hasStopDuration; the client
//     sets m_nStopDuration = value + get_update_time() and m_bEscortStop = 1 — a
//     timed stop that auto-resumes.
//   - stopIndefinitely : bool  — Decode1; sets m_nStopDuration = 0 and
//     m_bEscortStop = 1 — the mob never auto-resumes.
//
// IDA basis: CMob::OnEscortFullPath — v92 @0x6374c0, v95 @0x643d90,
// jms @0x6efa01. A 2-waypoint path with no attr==2 entry is 9×Decode4
// (count + oldDestX + oldDestY + 2×(x,y,attr) + currentDestIndex), then
// Decode1 [+Decode4] + Decode1. v92 + v95 + jms only — the escort family is
// absent in v83/v84/v87.
//
// packet-audit:fname CMob::OnEscortFullPath
type MobEscortFullPath struct {
	oldDestX         int32
	oldDestY         int32
	waypoints        []MobEscortWaypoint
	currentDestIndex int32
	hasStopDuration  bool
	stopDuration     int32
	stopIndefinitely bool
}

func NewMobEscortFullPath(oldDestX int32, oldDestY int32, waypoints []MobEscortWaypoint, currentDestIndex int32, hasStopDuration bool, stopDuration int32, stopIndefinitely bool) MobEscortFullPath {
	return MobEscortFullPath{
		oldDestX:         oldDestX,
		oldDestY:         oldDestY,
		waypoints:        waypoints,
		currentDestIndex: currentDestIndex,
		hasStopDuration:  hasStopDuration,
		stopDuration:     stopDuration,
		stopIndefinitely: stopIndefinitely,
	}
}

func (m MobEscortFullPath) OldDestX() int32                { return m.oldDestX }
func (m MobEscortFullPath) OldDestY() int32                { return m.oldDestY }
func (m MobEscortFullPath) Waypoints() []MobEscortWaypoint { return m.waypoints }
func (m MobEscortFullPath) CurrentDestIndex() int32        { return m.currentDestIndex }
func (m MobEscortFullPath) HasStopDuration() bool          { return m.hasStopDuration }
func (m MobEscortFullPath) StopDuration() int32            { return m.stopDuration }
func (m MobEscortFullPath) StopIndefinitely() bool         { return m.stopIndefinitely }
func (m MobEscortFullPath) Operation() string              { return MobEscortFullPathWriter }
func (m MobEscortFullPath) String() string {
	return fmt.Sprintf("oldDest [%d, %d], waypoints [%d], currentDestIndex [%d], hasStopDuration [%t], stopDuration [%d], stopIndefinitely [%t]",
		m.oldDestX, m.oldDestY, len(m.waypoints), m.currentDestIndex, m.hasStopDuration, m.stopDuration, m.stopIndefinitely)
}

func (m MobEscortFullPath) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(int32(len(m.waypoints)))
		w.WriteInt32(m.oldDestX)
		w.WriteInt32(m.oldDestY)
		for _, wp := range m.waypoints {
			w.WriteInt32(wp.x)
			w.WriteInt32(wp.y)
			w.WriteInt32(wp.attr)
			if wp.attr == 2 {
				w.WriteInt32(wp.stopDuration)
			}
		}
		w.WriteInt32(m.currentDestIndex)
		w.WriteBool(m.hasStopDuration)
		if m.hasStopDuration {
			w.WriteInt32(m.stopDuration)
		}
		w.WriteBool(m.stopIndefinitely)
		return w.Bytes()
	}
}

func (m *MobEscortFullPath) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadInt32()
		m.oldDestX = r.ReadInt32()
		m.oldDestY = r.ReadInt32()
		m.waypoints = make([]MobEscortWaypoint, 0, count)
		for i := int32(0); i < count; i++ {
			var wp MobEscortWaypoint
			wp.x = r.ReadInt32()
			wp.y = r.ReadInt32()
			wp.attr = r.ReadInt32()
			if wp.attr == 2 {
				wp.stopDuration = r.ReadInt32()
			}
			m.waypoints = append(m.waypoints, wp)
		}
		m.currentDestIndex = r.ReadInt32()
		m.hasStopDuration = r.ReadBool()
		if m.hasStopDuration {
			m.stopDuration = r.ReadInt32()
		}
		m.stopIndefinitely = r.ReadBool()
	}
}
