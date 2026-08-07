package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// hasEnabledFlag reports whether the client reads the trailing bEnabled byte.
//
// The byte was introduced between v48 and v61. IDA: v48 CNpcPool::OnNpcEnterField
// @0x56d527 reads Decode4(id) then Decode4(template) and delegates to CNpc::Init
// (sub_566A30 @0x566a30), whose body reads exactly six fields — Decode2 @0x566a64,
// Decode2 @0x566a72, Decode1 @0x566a9a, Decode2 @0x566aaa, Decode2 @0x566aca,
// Decode2 @0x566ad8 — and then no further CInPacket call: 8 reads total. v61
// CNpc::Init @0x5e7bef, v72 @0x63d7c1 and v79 @0x660154 all read those same six
// plus a trailing Decode1: 9 reads total, matching v83 @0x6d9993, v87 @0x716fd5
// and v95 @0x679680.
//
// Writing the byte to v48 left one unread byte at the end of every NPC spawn.
func hasEnabledFlag(ctx context.Context) bool {
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		// Tenant-less contexts reach here only via the ResolveCode 99 sentinel
		// path (npc.NpcControllerGrantBody with an unmigrated config), which
		// asserts on the leading byte and must not panic. Keep the majority
		// shape: every version except GMS below 61 carries the byte.
		return true
	}
	return !t.IsRegion("GMS") || t.MajorAtLeast(61)
}

const NpcSpawnWriter = "SpawnNPC"

// packet-audit:fname CNpcPool::OnNpcEnterField
type Spawn struct {
	id       uint32
	template uint32
	x        int16
	cy       int16
	f        int32
	fh       uint16
	rx0      int16
	rx1      int16
}

func NewNpcSpawn(id uint32, template uint32, x int16, cy int16, f int32, fh uint16, rx0 int16, rx1 int16) Spawn {
	return Spawn{id: id, template: template, x: x, cy: cy, f: f, fh: fh, rx0: rx0, rx1: rx1}
}

func (m Spawn) Operation() string { return NpcSpawnWriter }
func (m Spawn) String() string {
	return fmt.Sprintf("id [%d], template [%d]", m.id, m.template)
}

func (m Spawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.template)
		w.WriteInt16(m.x)
		w.WriteInt16(m.cy)
		if m.f == 1 {
			w.WriteByte(0)
		} else {
			w.WriteByte(1)
		}
		w.WriteShort(m.fh)
		w.WriteInt16(m.rx0)
		w.WriteInt16(m.rx1)
		if hasEnabledFlag(ctx) {
			w.WriteByte(1) // bEnabled
		}
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.template = r.ReadUint32()
		m.x = r.ReadInt16()
		m.cy = r.ReadInt16()
		fByte := r.ReadByte()
		if fByte == 0 {
			m.f = 1
		} else {
			m.f = 0
		}
		m.fh = r.ReadUint16()
		m.rx0 = r.ReadInt16()
		m.rx1 = r.ReadInt16()
		if hasEnabledFlag(ctx) {
			_ = r.ReadByte() // bEnabled
		}
	}
}
