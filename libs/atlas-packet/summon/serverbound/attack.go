package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// AttackTarget is one damaged monster decoded from a summon ATTACK packet.
type AttackTarget struct {
	monsterOid uint32
	templateId uint32
	damage     uint32
	delay      int16
}

func (t AttackTarget) MonsterOid() uint32 { return t.monsterOid }
func (t AttackTarget) TemplateId() uint32 { return t.templateId }
func (t AttackTarget) Damage() uint32     { return t.damage }
func (t AttackTarget) Delay() int16       { return t.delay }

const SummonAttackHandle = "SummonAttackHandle"

// Attack is the client -> server summon ATTACK packet, decoded from the real
// client SEND site CSummoned::TryDoingAttackManual. Three structurally distinct
// layouts exist; all were confirmed at ASM level:
//
//	v83  (GMS < 87)   sub_7A4D42, send block @0x7a57dc, op 0xB0 — LEAN, no anti-hack envelope
//	v87  (GMS 87..94) @0x7f6666, send block @0x7f8a..,  op 0xBC — envelope + crc, NO repeatSkillPoint
//	v95  (GMS >= 95)  @0x751240, send block @0x75226b,  op 0xD0 — envelope + crc + repeatSkillPoint
//
// v83 header (lean):
//	Encode4 summonId        ; owner cid [obj+0xAC]  (0x7a57f1)
//	Encode4 updateTime      ; get_update_time()     (0x7a57ff)
//	Encode1 action|left     ; (left<<7)|action&0x7F (0x7a5814)
//	Encode1 count           ; mob count             (0x7a5820)
//	Encode2 userX, userY, summonX, summonY          (0x7a5839..0x7a587e)
//
// v87/v95 header (anti-hack envelope) — drInfo/dwKey/crc32 are read at exact
// widths to keep the cursor aligned; the server does NOT validate them:
//	Encode4 summonId        ; v87 = cid [obj+0xAC] (0x7f8a..); v95 = m_dwSummonedID (0x752287)
//	Encode4 ~drInfo[0]
//	Encode4 ~drInfo[1]
//	Encode4 updateTime
//	Encode4 ~drInfo[2]
//	Encode4 ~drInfo[3]
//	Encode1 action|left
//	Encode4 dwKey
//	Encode4 crc32
//	Encode1 count
//	Encode2 userX, userY, summonX, summonY
//	Encode4 repeatSkillPoint  ; v95 ONLY (0x752450); absent in v87
//
// Per-target block (26 bytes, identical across versions):
//	Encode4 mobOid                                  (v83 @0x7a58aa)
//	Encode4 templateId                              (v83 @0x7a58dc)
//	Encode1 hitAction                               (v83 @0x7a58ea)
//	Encode1 foreAction|left                         (v83 @0x7a5905)
//	Encode1 frameIdx                                (v83 @0x7a5913)
//	Encode1 calcDamageStatIndex                     (v83 @0x7a5923)
//	Encode2 curX, Encode2 curY                      (v83 @0x7a5939/0x7a5950)
//	Encode2 hitX, Encode2 hitY                      (v83 @0x7a5966/0x7a597d)
//	Encode2 tDelay                                  (v83 @0x7a598c)
//	Encode4 damage                                  (v83 @0x7a5997)
//
// Trailer: Encode4 skillCRC (v83 @0x7a59bd).
//
// Summon identity: v83/v87 carry the owner cid; v95 carries m_dwSummonedID. The
// value is exposed via SummonId(); the channel handler reconciles cid-vs-id
// against the sender's owned summons.
type Attack struct {
	summonId  uint32
	direction byte
	targets   []AttackTarget
}

func (m Attack) SummonId() uint32        { return m.summonId }
func (m Attack) Direction() byte         { return m.direction }
func (m Attack) Targets() []AttackTarget { return m.targets }

func (m Attack) Operation() string { return SummonAttackHandle }

func (m Attack) String() string {
	return fmt.Sprintf("summonId [%d], direction [%d], targets [%d]", m.summonId, m.direction, len(m.targets))
}

// hasAntiHackEnvelope reports whether the version emits the drInfo/dwKey/crc32
// anti-hack envelope (GMS >= 87). v83/v84 send the lean layout.
func hasAntiHackEnvelope(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(87)
}

// hasRepeatSkillPoint reports whether the version emits the trailing
// repeatSkillPoint int after the position block (GMS >= 95).
func hasRepeatSkillPoint(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(95)
}

func (m Attack) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.summonId)
		if hasAntiHackEnvelope(t) {
			w.Skip(4) // ~drInfo[0]
			w.Skip(4) // ~drInfo[1]
			w.Skip(4) // updateTime
			w.Skip(4) // ~drInfo[2]
			w.Skip(4) // ~drInfo[3]
			w.WriteByte(m.direction)
			w.Skip(4) // dwKey
			w.Skip(4) // crc32
			w.WriteByte(byte(len(m.targets)))
			w.Skip(8) // user x,y + summon x,y
			if hasRepeatSkillPoint(t) {
				w.Skip(4) // repeatSkillPoint (v95 only)
			}
		} else {
			w.Skip(4) // updateTime
			w.WriteByte(m.direction)
			w.WriteByte(byte(len(m.targets)))
			w.Skip(8) // user x,y + summon x,y
		}
		for _, tg := range m.targets {
			w.WriteInt(tg.monsterOid)
			w.WriteInt(tg.templateId)
			w.Skip(4) // hitAction, foreAction|left, frameIdx, calcDamageStatIndex
			w.Skip(8) // curX, curY, hitX, hitY
			w.WriteInt16(tg.delay)
			w.WriteInt(tg.damage)
		}
		w.Skip(4) // skillCRC
		return w.Bytes()
	}
}

func (m *Attack) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.summonId = r.ReadUint32()
		var count int
		if hasAntiHackEnvelope(t) {
			r.Skip(4) // ~drInfo[0]
			r.Skip(4) // ~drInfo[1]
			r.Skip(4) // updateTime
			r.Skip(4) // ~drInfo[2]
			r.Skip(4) // ~drInfo[3]
			m.direction = r.ReadByte()
			r.Skip(4) // dwKey
			r.Skip(4) // crc32
			count = int(r.ReadByte())
			r.Skip(8) // user x,y + summon x,y
			if hasRepeatSkillPoint(t) {
				r.Skip(4) // repeatSkillPoint (v95 only)
			}
		} else {
			r.Skip(4) // updateTime
			m.direction = r.ReadByte()
			count = int(r.ReadByte())
			r.Skip(8) // user x,y + summon x,y
		}
		m.targets = make([]AttackTarget, 0, count)
		for i := 0; i < count; i++ {
			monsterOid := r.ReadUint32()
			templateId := r.ReadUint32()
			r.Skip(4) // hitAction(1), foreAction|left(1), frameIdx(1), calcDamageStatIndex(1)
			r.Skip(8) // curX(2), curY(2), hitX(2), hitY(2)
			delay := r.ReadInt16()
			damage := r.ReadUint32()
			m.targets = append(m.targets, AttackTarget{monsterOid: monsterOid, templateId: templateId, damage: damage, delay: delay})
		}
		r.Skip(4) // skillCRC trailer
	}
}
