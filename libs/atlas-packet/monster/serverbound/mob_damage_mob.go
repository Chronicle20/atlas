package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobDamageMobHandle = "MobDamageMob"

// MobDamageMob is the serverbound MOB_DAMAGE_MOB packet (CMob::SetDamagedByMob):
// the controller reports that one mob hit another mob (e.g. a charmed/escort mob
// taking body damage from a hostile mob) so the server can apply the damage.
//
// Byte layout (IDA-verified, identical across all 5 versions — the COutPacket
// build site in CMob::SetDamagedByMob):
//   - attackerMobId : uint32 — mob id of the attacker (GetMobID(nDamage))
//   - characterId   : uint32 — the controlling user's character id (CWvsContext)
//   - mobId         : uint32 — mob id of the victim (GetMobID(this))
//   - attackIndex   : byte   — which attack template entry struck (nAttackIdx, vx)
//   - damage        : uint32 — computed P/M damage applied to the victim
//   - reflect       : byte   — dir<0 flag (Encode1(vy < 0)); knock/reflect side
//   - x             : uint16 — body-rect x centre of the damage effect
//   - y             : uint16 — body-rect y centre of the damage effect
//
// IDA basis: CMob::SetDamagedByMob — v83 @0x670c63 (opcode 0xC2), v87 @0x6abd95,
// v95 @0x64b260 (opcode 0xE9):
//
//	COutPacket(op); Encode4(GetMobID(attacker)); Encode4(characterId);
//	Encode4(GetMobID(this)); Encode1(nAttackIdx); Encode4(damage);
//	Encode1(nDir<0); Encode2(xCenter); Encode2(yCenter); SendPacket
//
// packet-audit:fname CMob::SetDamagedByMob
type MobDamageMob struct {
	attackerMobId uint32
	characterId   uint32
	mobId         uint32
	attackIndex   byte
	damage        uint32
	reflect       byte
	x             uint16
	y             uint16
}

func (m MobDamageMob) AttackerMobId() uint32 { return m.attackerMobId }
func (m MobDamageMob) CharacterId() uint32   { return m.characterId }
func (m MobDamageMob) MobId() uint32         { return m.mobId }
func (m MobDamageMob) AttackIndex() byte     { return m.attackIndex }
func (m MobDamageMob) Damage() uint32        { return m.damage }
func (m MobDamageMob) Reflect() byte         { return m.reflect }
func (m MobDamageMob) X() uint16             { return m.x }
func (m MobDamageMob) Y() uint16             { return m.y }
func (m MobDamageMob) Operation() string     { return MobDamageMobHandle }
func (m MobDamageMob) String() string {
	return fmt.Sprintf("attackerMobId [%d], characterId [%d], mobId [%d], attackIndex [%d], damage [%d], reflect [%d], x [%d], y [%d]",
		m.attackerMobId, m.characterId, m.mobId, m.attackIndex, m.damage, m.reflect, m.x, m.y)
}

func (m MobDamageMob) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.attackerMobId)
		w.WriteInt(m.characterId)
		w.WriteInt(m.mobId)
		w.WriteByte(m.attackIndex)
		w.WriteInt(m.damage)
		w.WriteByte(m.reflect)
		w.WriteShort(m.x)
		w.WriteShort(m.y)
		return w.Bytes()
	}
}

func (m *MobDamageMob) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attackerMobId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.mobId = r.ReadUint32()
		m.attackIndex = r.ReadByte()
		m.damage = r.ReadUint32()
		m.reflect = r.ReadByte()
		m.x = r.ReadUint16()
		m.y = r.ReadUint16()
	}
}
