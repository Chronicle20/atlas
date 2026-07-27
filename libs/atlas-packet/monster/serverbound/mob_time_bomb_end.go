package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobTimeBombEndHandle = "MobTimeBombEnd"

// MobTimeBombEnd is the serverbound MOB_TIME_BOMB_END packet
// (CMob::UpdateTimeBomb): when a mob's time-bomb timer elapses the controller
// reports the detonation so the server can apply its effect to the local user.
//
// Byte layout (IDA-verified — CMob::UpdateTimeBomb COutPacket build site):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//   - (boss only) bossX : uint32 — boss body-rect x centre
//   - (boss only) bossY : uint32 — boss body-rect y centre
//   - localX : uint32 — local user position x
//   - localY : uint32 — local user position y
//
// The bossX/bossY pair is present ONLY when the mob is a boss
// (_ZtlSecureFuse(m_pTemplate->bBoss)); this is a data-dependent conditional, not
// a version branch — the controller knows the mob's boss flag, so it is carried on
// the model as `boss` (off the wire) and both Encode and Decode honour it.
//
// IDA basis: CMob::UpdateTimeBomb — v95 @0x643c30 (opcode 0xEB), jms @0x6ef8f8
// (opcode 0xCA):
//
//	COutPacket(op); Encode4(SecureFuse(m_dwMobID));
//	if (bBoss) { Encode4(xCenter); Encode4(yCenter); }
//	Encode4(localUser.x); Encode4(localUser.y); SendPacket
//
// VERSION note: in v83/v84/v87 this send is INLINED into CMob::Update (no
// standalone UpdateTimeBomb function), so those versions have no discrete pinnable
// symbol (see structures/applicability.md / RESUME-STATE.md). The wire shape is the
// same; the route still lands so the handler is wired, but evidence is pinned only
// for v95/jms where the standalone function exists.
//
// packet-audit:fname CMob::UpdateTimeBomb
type MobTimeBombEnd struct {
	boss   bool
	mobCrc uint32
	bossX  uint32
	bossY  uint32
	localX uint32
	localY uint32
}

func (m MobTimeBombEnd) Boss() bool        { return m.boss }
func (m MobTimeBombEnd) MobCrc() uint32    { return m.mobCrc }
func (m MobTimeBombEnd) BossX() uint32     { return m.bossX }
func (m MobTimeBombEnd) BossY() uint32     { return m.bossY }
func (m MobTimeBombEnd) LocalX() uint32    { return m.localX }
func (m MobTimeBombEnd) LocalY() uint32    { return m.localY }
func (m MobTimeBombEnd) Operation() string { return MobTimeBombEndHandle }
func (m MobTimeBombEnd) String() string {
	return fmt.Sprintf("boss [%t], mobCrc [%d], bossX [%d], bossY [%d], localX [%d], localY [%d]",
		m.boss, m.mobCrc, m.bossX, m.bossY, m.localX, m.localY)
}

func (m MobTimeBombEnd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		if m.boss {
			w.WriteInt(m.bossX)
			w.WriteInt(m.bossY)
		}
		w.WriteInt(m.localX)
		w.WriteInt(m.localY)
		return w.Bytes()
	}
}

func (m *MobTimeBombEnd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
		if m.boss {
			m.bossX = r.ReadUint32()
			m.bossY = r.ReadUint32()
		}
		m.localX = r.ReadUint32()
		m.localY = r.ReadUint32()
	}
}
