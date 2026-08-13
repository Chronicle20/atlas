package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const DragonSpawnWriter = "DragonSpawn"

// DragonSpawn is the server -> client SPAWN_DRAGON packet.
//
// Wire: int ownerCharacterId, int x, int y, byte stance, short <discarded>,
// [short jobId — version-gated, see spawnHasJobId].
//
// The leading ownerCharacterId is consumed by CUserPool::OnUserCommonPacket
// (GMS v95.0 @0x94cdb0), which resolves the CUser and only then routes the
// dragon family. The dragon has no wire identity of its own.
//
// TRAPS, both load-bearing:
//   - x and y are FOUR bytes each (Decode4), not the 2-byte coordinates used by
//     every other entity in the protocol.
//   - the short between stance and jobId is read by the client and thrown away
//     (the Decode2 return value is never assigned) on EVERY version, including
//     v83 where jobId itself is absent. It must still be written or, on
//     versions that carry jobId, jobId is read from the wrong offset.
//
// jobId is NOT uniform across versions (spawnHasJobId): GMS v83's
// CDragon::OnCreated (@0x4fe502) reads x, y, stance, then the discarded short
// and stops — 11 bytes total, no jobId read. This matches the domain: Evan
// does not exist in v83 (v84 is the first job table binding 2200-2218), so
// v83's CDragon has no job code to send. GMS v84's spawn arm sub_506F85
// (@0x506f85) adds `this[63] = (unsigned __int16)CInPacket::Decode2(v3)`
// after the same four reads; JMS185's CDragon::OnCreated (@0x52edd3) adds
// `*(this + 71) = CInPacket::Decode2(iPacket)` identically; GMS v95 also
// carries jobId (already verified). Both present-jobId versions total 13
// bytes; v83 totals 11. v87 and v92 are not yet independently verified but
// sit between v84 and v95.
//
// packet-audit:fname CDragon::OnCreated
type DragonSpawn struct {
	ownerCharacterId uint32
	x                int32
	y                int32
	stance           byte
	jobId            uint16
}

func NewDragonSpawn(ownerCharacterId uint32, x int32, y int32, stance byte, jobId uint16) DragonSpawn {
	return DragonSpawn{ownerCharacterId: ownerCharacterId, x: x, y: y, stance: stance, jobId: jobId}
}

func (m DragonSpawn) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonSpawn) X() int32                 { return m.x }
func (m DragonSpawn) Y() int32                 { return m.y }
func (m DragonSpawn) Stance() byte             { return m.stance }
func (m DragonSpawn) JobId() uint16            { return m.jobId }
func (m DragonSpawn) Operation() string        { return DragonSpawnWriter }
func (m DragonSpawn) String() string {
	return fmt.Sprintf("ownerCharacterId [%d], x [%d], y [%d], stance [%d], jobId [%d]", m.ownerCharacterId, m.x, m.y, m.stance, m.jobId)
}

// spawnHasJobId reports whether this version's CDragon::OnCreated reads a
// trailing Decode2 jobId after the discarded short. IDA-grounded: ABSENT in
// GMS v83 (0x4fe502 — the read sequence stops after the discarded short);
// PRESENT in GMS v84 (sub_506F85 @0x506f85, `this[63] = Decode2()`), JMS185
// (0x52edd3, `*(this + 71) = Decode2()`), and GMS v95 (already verified).
// Matches the domain: Evan (and its dragon) does not exist before GMS v84.
func spawnHasJobId(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.IsRegion("JMS")
}

func (m DragonSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0) // client decodes and discards; omitting it misaligns jobId
		if spawnHasJobId(ctx) {
			w.WriteShort(m.jobId)
		}
		return w.Bytes()
	}
}

func (m *DragonSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
		m.stance = r.ReadByte()
		_ = r.ReadUint16() // discarded by the client (see Encode)
		if spawnHasJobId(ctx) {
			m.jobId = r.ReadUint16()
		}
	}
}
