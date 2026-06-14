package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldDamageMobHandle = "FieldDamageMob"

// FieldDamageMob is the serverbound FIELD_DAMAGE_MOB packet, built by CMob::Update
// when a controlled mob takes environmental ("field") damage (lava/poison-floor
// obstacle ticks) and the controller reports the applied damage to the server.
//
// Byte layout (IDA-verified, identical across all 5 versions — the field-damage
// COutPacket build site inside CMob::Update):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//   - damage : uint32 — the field damage amount applied this tick
//
// IDA basis: CMob::Update field-damage send site — v83 @0x667d39 (opcode 0xBF),
// v87 @0x6a24a5, v95 @0x654fc9 (opcode 0xE6), jms @0x6e457c:
//
//	COutPacket(op); Encode4(SecureFuse(m_dwMobID)); Encode4(nFieldDamage); SendPacket
//
// v84 emits this at opcode 0xC4 @0x67dd33 (registry was csv-import-stale at 0xBF).
type FieldDamageMob struct {
	mobCrc uint32
	damage uint32
}

func (m FieldDamageMob) MobCrc() uint32    { return m.mobCrc }
func (m FieldDamageMob) Damage() uint32    { return m.damage }
func (m FieldDamageMob) Operation() string { return FieldDamageMobHandle }
func (m FieldDamageMob) String() string {
	return fmt.Sprintf("mobCrc [%d], damage [%d]", m.mobCrc, m.damage)
}

func (m FieldDamageMob) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		w.WriteInt(m.damage)
		return w.Bytes()
	}
}

func (m *FieldDamageMob) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
		m.damage = r.ReadUint32()
	}
}
