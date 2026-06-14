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
//	v83  (GMS == 83)  sub_7A4D42, send block @0x7a57dc, op 0xB0 — LEAN, no anti-hack envelope
//	v84  (GMS 84..86) sub_7C99CF, send block @0x7cafcd, op 0xB5 — envelope + crc, NO repeatSkillPoint
//	v87  (GMS 87..94) @0x7f6666, send block @0x7f8a..,  op 0xBC — envelope + crc, NO repeatSkillPoint
//	v95  (GMS >= 95)  @0x751240, send block @0x75226b,  op 0xD0 — envelope + crc + repeatSkillPoint
//
// IMPORTANT (task-088, v84 IDA audit): the anti-hack envelope is NOT v87+ only —
// v84 ALSO sends drInfo/dwKey/crc32 (GMS_v84.1 CSummoned::TryDoingAttackManual
// send block @0x7cafcd: COutPacket(181) + Encode4 cid + Encode4 ~drInfo0
// @0x7caffc + Encode4 ~drInfo1 @0x7cb010 + Encode4 updateTime @0x7cb021 +
// Encode4 ~drInfo2 @0x7cb035 + Encode4 ~drInfo3 @0x7cb049 + Encode1 action
// @0x7cb069 + Encode4 dwKey @0x7cb0c5 + Encode4 crc32 @0x7cb0ec + Encode1 count
// @0x7cb0fd + Encode2 positions + per-target + Encode4 skillCRC @0x7cb485; NO
// repeatSkillPoint). Only v83 is lean. The envelope gate is therefore
// MajorAtLeast(84), not MajorAtLeast(87).
//
// v83 header (lean):
//
//	Encode4 summonId        ; owner cid [obj+0xAC]  (0x7a57f1)
//	Encode4 updateTime      ; get_update_time()     (0x7a57ff)
//	Encode1 action|left     ; (left<<7)|action&0x7F (0x7a5814)
//	Encode1 count           ; mob count             (0x7a5820)
//	Encode2 userX, userY, summonX, summonY          (0x7a5839..0x7a587e)
//
// v87/v95 header (anti-hack envelope) — drInfo/dwKey/crc32 are read at exact
// widths to keep the cursor aligned; the server does NOT validate them:
//
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
//
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
// anti-hack envelope. Present on GMS >= 84 (only v83 sends the lean layout; v84
// already carries the envelope — IDA-confirmed, GMS_v84.1 send block @0x7cafcd)
// AND on JMS v185. The jms185 summon manual-attack send (TryDoingAttackManual,
// inlined into CSummoned::Update via sub_824A81@0x824a81) calls DR_check@0x826202
// and stores the resulting _DR_INFO block (var_20@0x8261fd) for the send — the
// same anti-hack envelope GMS gained at v84, IDB-confirmed present in jms185.
// The COutPacket(0xB3) emit itself is behind the jms185 anti-tamper VM
// (jmp loc_DE90B8@0x82620f -> loc_D21897), so the field order below mirrors the
// v95 PDB-clean send (CSummoned::TryDoingAttackManual@0x751240); the envelope's
// PRESENCE is proven by the DR_check call.
func hasAntiHackEnvelope(t tenant.Model) bool {
	if t.IsRegion("JMS") {
		return t.MajorAtLeast(185)
	}
	return t.IsRegion("GMS") && t.MajorAtLeast(84)
}

// hasRepeatSkillPoint reports whether the version emits the trailing
// repeatSkillPoint int after the position block. Present on GMS >= 95
// (CUserLocal::GetRepeatSkillPoint v95@0x748e50 -> Encode4@0x752450) AND on JMS
// v185. repeatSkillPoint is a permanent post-v95 addition to the summon-attack
// envelope; JMS v185 (v185 >> v95 in the shared GMS/JMS code lineage) inherits
// it. The jms185 send is VM-obfuscated so the field cannot be read directly; its
// presence is inferred from the build lineage (the envelope itself is confirmed
// via DR_check). The decoder skips it at its exact width to stay aligned.
func hasRepeatSkillPoint(t tenant.Model) bool {
	if t.IsRegion("JMS") {
		return t.MajorAtLeast(185)
	}
	return t.IsRegion("GMS") && t.MajorAtLeast(95)
}

func (m Attack) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// NOTE: every field below is written at its exact client-read width via an
		// explicit zero-write rather than w.Skip(n). The bytes are byte-identical
		// to the prior Skip(n) form (Skip zero-fills) — this is a wire-neutral
		// rewrite whose only purpose is to make the per-field shape visible to the
		// packet-audit flat-diff analyzer (which cannot model Skip). The
		// anti-hack envelope fields (drInfo/dwKey/crc32/updateTime/positions) are
		// consumed by the decoder at the same widths but never validated.
		w.WriteInt(m.summonId)
		if hasAntiHackEnvelope(t) {
			w.WriteInt(0) // ~drInfo[0]   @0x75229b
			w.WriteInt(0) // ~drInfo[1]   @0x7522af
			w.WriteInt(0) // updateTime   @0x7522c0
			w.WriteInt(0) // ~drInfo[2]   @0x7522d4
			w.WriteInt(0) // ~drInfo[3]   @0x7522e8
			w.WriteByte(m.direction)
			w.WriteInt(0) // dwKey        @0x752325
			w.WriteInt(0) // crc32        @0x75234c
			w.WriteByte(byte(len(m.targets)))
			w.WriteInt16(0) // userX       @0x7523a5
			w.WriteInt16(0) // userY       @0x7523dd
			w.WriteInt16(0) // summonX     @0x75240a
			w.WriteInt16(0) // summonY     @0x752438
			if hasRepeatSkillPoint(t) {
				w.WriteInt(0) // repeatSkillPoint (v95 only) @0x752450
			}
		} else {
			w.WriteInt(0) // updateTime
			w.WriteByte(m.direction)
			w.WriteByte(byte(len(m.targets)))
			w.WriteInt16(0) // userX
			w.WriteInt16(0) // userY
			w.WriteInt16(0) // summonX
			w.WriteInt16(0) // summonY
		}
		for _, tg := range m.targets {
			w.WriteInt(tg.monsterOid) // mob[i].mobId       @0x7524ac
			w.WriteInt(tg.templateId) // mob[i].templateId  @0x7524cc
			w.WriteByte(0)            // hitAction          @0x7524e2
			w.WriteByte(0)            // foreAction|left    @0x75250c
			w.WriteByte(0)            // frameIdx           @0x752522
			w.WriteByte(0)            // calcDamageStatIdx  @0x75253b
			w.WriteInt16(0)           // curX (hitX)        @0x75256c
			w.WriteInt16(0)           // curY (hitY)        @0x7525a0
			w.WriteInt16(0)           // hitX (posX)        @0x7525d3
			w.WriteInt16(0)           // hitY (posY)        @0x752607
			w.WriteInt16(tg.delay)    // tDelay             @0x75261d
			w.WriteInt(tg.damage)     // damage             @0x752632
		}
		w.WriteInt(0) // skillCRC @0x75266f
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
