package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// mobBlock builds one 26-byte per-target block as the client SEND emits it:
// mobOid(4) templateId(4) hitAction(1) foreAction|left(1) frameIdx(1)
// calcDamageStatIndex(1) curX(2) curY(2) hitX(2) hitY(2) tDelay(2) damage(4).
func mobBlock(mobOid, templateId uint32, tDelay int16, damage uint32) []byte {
	b := make([]byte, 0, 26)
	b = append(b, le32(mobOid)...)
	b = append(b, le32(templateId)...)
	b = append(b, 0x01, 0x02, 0x03, 0x04) // hitAction, foreAction|left, frameIdx, calcDamageStatIndex
	b = append(b, le16(11)...)            // curX
	b = append(b, le16(22)...)            // curY
	b = append(b, le16(33)...)            // hitX
	b = append(b, le16(44)...)            // hitY
	b = append(b, le16(uint16(tDelay))...) // tDelay
	b = append(b, le32(damage)...)         // damage
	return b
}

func le16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }
func le32(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

// mobBlockV79 builds one 22-byte per-target block as the v79 client SEND emits
// it: mobOid(4) — NO templateId — hitAction(1) foreAction|left(1) frameIdx(1)
// calcDamageStatIndex(1) curX(2) curY(2) hitX(2) hitY(2) tDelay(2) damage(4).
func mobBlockV79(mobOid uint32, tDelay int16, damage uint32) []byte {
	b := make([]byte, 0, 22)
	b = append(b, le32(mobOid)...)
	b = append(b, 0x01, 0x02, 0x03, 0x04) // hitAction, foreAction|left, frameIdx, calcDamageStatIndex
	b = append(b, le16(11)...)            // curX
	b = append(b, le16(22)...)            // curY
	b = append(b, le16(33)...)            // hitX
	b = append(b, le16(44)...)            // hitY
	b = append(b, le16(uint16(tDelay))...) // tDelay
	b = append(b, le32(damage)...)         // damage
	return b
}

// TestSummonAttackByteV72 pins the gms_v72 SUMMON_ATTACK (op 170) serverbound
// send. IDA: the op-170 sender sub_6E787F @0x6e787f (GMS_v72.1_U_DEVM.exe, port
// 13339) — CSummoned::TryDoingAttackManual is unnamed in the v72 IDB; the send
// block was located by the COutPacket(170) harvest. The v72 frame is the
// legacy-lean layout, IDENTICAL to v79 EXCEPT it has NO trailing skillCRC int —
// the send closes at Encode2 summonY@0x6e841a then SendPacket@0x6e8429 (disasm
// confirmed: no further Encode4). skillCRC was added between v72 and v79.
//
//	Encode4 summonId               @0x6e82c4
//	Encode4 updateTime             @0x6e82d2
//	Encode1 action|left            @0x6e82eb ((v129<<7)|v123&0x7F)
//	Encode1 count                  @0x6e82f6 (HitMobInRect)
//	per-target (25 bytes, mob OID only — NO templateId):
//	  Encode4 mobOID               @0x6e8327
//	  Encode1 hitAction            @0x6e8335
//	  Encode1 foreAction|left      @0x6e8350
//	  Encode1 frameIdx             @0x6e835e
//	  Encode1 calcDamageStatIndex  @0x6e836e
//	  Encode2 curX,curY,hitX,hitY  @0x6e8384..0x6e83c8
//	  Encode2 tDelay               @0x6e83d7
//	  Encode4 damage               @0x6e83e2
//	Encode2 summonX                @0x6e8406
//	Encode2 summonY                @0x6e841a
//	(NO skillCRC — SendPacket@0x6e8429)
//
// DECODE fixture (like the v79 sibling): hand-build a real-shaped body with NO
// trailing skillCRC and assert a clean cursor. hasSkillCrcTrailer(GMS,72)=false.
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v72 ida=0x6e787f
func TestSummonAttackByteV72(t *testing.T) {
	body := []byte{}
	body = append(body, le32(1000005)...) // summonId
	body = append(body, le32(123456)...)   // updateTime
	body = append(body, 0x83)              // action|left (left bit + action 3)
	body = append(body, 0x02)              // count = 2
	body = append(body, mobBlockV79(2000001, 100, 1234)...)
	body = append(body, mobBlockV79(2000002, -50, 5678)...)
	body = append(body, le16(510)...) // summonX (after targets)
	body = append(body, le16(590)...) // summonY
	// NO skillCRC on v72

	ctx := test.CreateContext("GMS", 72, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	if m.SummonId() != 1000005 {
		t.Errorf("summonId = %d, want 1000005", m.SummonId())
	}
	if m.Direction() != 0x83 {
		t.Errorf("direction = %#x, want 0x83", m.Direction())
	}
	if len(m.Targets()) != 2 {
		t.Fatalf("targets len = %d, want 2", len(m.Targets()))
	}
	t0 := m.Targets()[0]
	if t0.MonsterOid() != 2000001 || t0.TemplateId() != 0 || t0.Damage() != 1234 || t0.Delay() != 100 {
		t.Errorf("target[0] = %+v, want oid=2000001 tmpl=0 dmg=1234 delay=100", t0)
	}
	t1 := m.Targets()[1]
	if t1.MonsterOid() != 2000002 || t1.TemplateId() != 0 || t1.Damage() != 5678 || t1.Delay() != -50 {
		t.Errorf("target[1] = %+v, want oid=2000002 tmpl=0 dmg=5678 delay=-50", t1)
	}
	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// TestSummonAttackByteV79 pins the gms_v79 SUMMON_ATTACK (op 0xAC) serverbound
// send. IDA: CSummoned::TryDoingAttackManual @0x71b522, send block
// COutPacket(172)@0x71bfe1 (GMS_v79_1_DEVM.exe, port 13340). The v79 layout is
// the legacy-lean frame: summonId + updateTime + action|left + count, then the
// per-target loop (mob OID only — NO templateId), then summonX + summonY, then
// skillCRC. There is NO anti-hack envelope and NO leading position block.
//
//	Encode4 summonId               @0x71bff6 (v130[42])
//	Encode4 updateTime             @0x71c004
//	Encode1 action|left            @0x71c019 ((left<<7)|attackType&0x7F)
//	Encode1 count                  @0x71c025
//	per-target:
//	  Encode4 mobOID               @0x71c051 (sub_4DC1C0(mob+380))
//	  Encode1 hitAction            @0x71c05f
//	  Encode1 foreAction|left      @0x71c07a
//	  Encode1 frameIdx             @0x71c088
//	  Encode1 calcDamageStatIndex  @0x71c098
//	  Encode2 curX,curY,hitX,hitY  @0x71c0ae..0x71c0f2
//	  Encode2 tDelay               @0x71c101
//	  Encode4 damage               @0x71c10c
//	Encode2 summonX                @0x71c130
//	Encode2 summonY                @0x71c144
//	Encode4 skillCRC               @0x71c15d
//
// This is a DECODE fixture (like the v83/v87 siblings): the encoder zero-fills
// the positions/crc/updateTime, so decoding a hand-built real-shaped body and
// asserting a clean cursor + correct mob oids/damage/delay proves the field
// alignment.
//
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v79 ida=0x71b522
func TestSummonAttackByteV79(t *testing.T) {
	body := []byte{}
	body = append(body, le32(1000005)...) // summonId
	body = append(body, le32(123456)...)   // updateTime
	body = append(body, 0x83)              // action|left (left bit + action 3)
	body = append(body, 0x02)              // count = 2
	body = append(body, mobBlockV79(2000001, 100, 1234)...)
	body = append(body, mobBlockV79(2000002, -50, 5678)...)
	body = append(body, le16(510)...)    // summonX (after targets)
	body = append(body, le16(590)...)    // summonY
	body = append(body, le32(0xABCD)...) // skillCRC

	ctx := test.CreateContext("GMS", 79, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	if m.SummonId() != 1000005 {
		t.Errorf("summonId = %d, want 1000005", m.SummonId())
	}
	if m.Direction() != 0x83 {
		t.Errorf("direction = %#x, want 0x83", m.Direction())
	}
	if len(m.Targets()) != 2 {
		t.Fatalf("targets len = %d, want 2", len(m.Targets()))
	}
	// v79 sends no templateId, so TemplateId() stays 0.
	t0 := m.Targets()[0]
	if t0.MonsterOid() != 2000001 || t0.TemplateId() != 0 || t0.Damage() != 1234 || t0.Delay() != 100 {
		t.Errorf("target[0] = %+v, want oid=2000001 tmpl=0 dmg=1234 delay=100", t0)
	}
	t1 := m.Targets()[1]
	if t1.MonsterOid() != 2000002 || t1.TemplateId() != 0 || t1.Damage() != 5678 || t1.Delay() != -50 {
		t.Errorf("target[1] = %+v, want oid=2000002 tmpl=0 dmg=5678 delay=-50", t1)
	}
	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// TestSummonAttackDecodeV83 decodes a real-shaped 2-target LEAN v83 attack send
// (no anti-hack envelope) and asserts the cursor ends clean with the right
// target mob oids + damages. Confirmed against CSummoned::TryDoingAttackManual
// send block @0x7a57dc.
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v83 ida=0x7a4d42
func TestSummonAttackDecodeV83(t *testing.T) {
	body := []byte{}
	body = append(body, le32(1000005)...) // summonId (= owner cid on v83)
	body = append(body, le32(123456)...)   // updateTime
	body = append(body, 0x83)              // action|left (left bit set, action 3)
	body = append(body, 0x02)              // count = 2
	body = append(body, le16(500)...)      // userX
	body = append(body, le16(600)...)      // userY
	body = append(body, le16(510)...)      // summonX
	body = append(body, le16(590)...)      // summonY
	body = append(body, mobBlock(2000001, 9300018, 100, 1234)...)
	body = append(body, mobBlock(2000002, 9300166, -50, 5678)...)
	body = append(body, le32(0xABCD)...)   // skillCRC

	ctx := test.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	assertAttack(t, &m, &reader, 1000005)
}

// TestSummonAttackDecodeV87 decodes a real-shaped 2-target v87 attack send: the
// anti-hack envelope (drInfo/dwKey/crc32) is present but there is NO trailing
// repeatSkillPoint. Confirmed against CSummoned::TryDoingAttackManual @0x7f6666
// (send block: COutPacket(0xBC) + summonId@0x7f7c7f + 4×drInfo + updateTime +
// action + dwKey + crc32 + count + 4×pos, then the per-target loop @0x7f7e6d with
// NO repeatSkillPoint, then skillCRC@0x7f811c).
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v87 ida=0x7f6666
func TestSummonAttackDecodeV87(t *testing.T) {
	body := envelopeHeader(1000005, false)
	body = append(body, mobBlock(2000001, 9300018, 100, 1234)...)
	body = append(body, mobBlock(2000002, 9300166, -50, 5678)...)
	body = append(body, le32(0xABCD)...) // skillCRC

	ctx := test.CreateContext("GMS", 87, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	assertAttack(t, &m, &reader, 1000005)
}

// TestSummonAttackDecodeV84 decodes a real-shaped 2-target v84 attack send. Unlike
// v83 (lean), v84 ALREADY carries the anti-hack envelope (drInfo/dwKey/crc32) with
// NO repeatSkillPoint — byte-for-byte the v87 layout. Confirmed against the
// GMS_v84.1 client SEND site CSummoned::TryDoingAttackManual sub_7C99CF, send
// block @0x7cafcd: COutPacket(181)@0x7cafcd + Encode4 cid@0x7cafe8 + 4×drInfo +
// updateTime@0x7cb021 + action@0x7cb069 + dwKey@0x7cb0c5 + crc32@0x7cb0ec +
// count@0x7cb0fd + 4×pos, then the per-target loop @0x7cb256 (NO repeatSkillPoint),
// then skillCRC@0x7cb485. This is why the Atlas envelope gate is MajorAtLeast(84).
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v84 ida=0x7c99cf
func TestSummonAttackDecodeV84(t *testing.T) {
	body := envelopeHeader(1000005, false)
	body = append(body, mobBlock(2000001, 9300018, 100, 1234)...)
	body = append(body, mobBlock(2000002, 9300166, -50, 5678)...)
	body = append(body, le32(0xABCD)...) // skillCRC

	ctx := test.CreateContext("GMS", 84, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	assertAttack(t, &m, &reader, 1000005)
}

// TestSummonAttackDecodeV95 decodes a real-shaped 2-target v95 attack send: the
// anti-hack envelope PLUS the trailing repeatSkillPoint int. Confirmed against
// CSummoned::TryDoingAttackManual @0x751240.
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=gms_v95 ida=0x751240
func TestSummonAttackDecodeV95(t *testing.T) {
	body := envelopeHeader(1000005, true)
	body = append(body, mobBlock(2000001, 9300018, 100, 1234)...)
	body = append(body, mobBlock(2000002, 9300166, -50, 5678)...)
	body = append(body, le32(0xABCD)...) // skillCRC

	ctx := test.CreateContext("GMS", 95, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	assertAttack(t, &m, &reader, 1000005)
}

// TestSummonAttackDecodeJMS185 decodes a real-shaped 2-target JMS185 attack send:
// the anti-hack envelope but NO trailing repeatSkillPoint — structurally the GMS
// v87 layout, NOT v95. JMS v185 corresponds to the GMS v87 era (the JMS/GMS code
// lines diverged; raw version numbers don't map — v185 ≈ v87 here, not v95). The
// jms185 manual-attack (TryDoingAttackManual inlined into CSummoned::Update via
// sub_824A81@0x824a81) calls DR_check@0x826202 and stages the _DR_INFO envelope
// (var_20@0x8261fd) for the COutPacket(0xB3) emit. The emit itself is behind the
// jms185 anti-tamper VM (jmp loc_DE90B8@0x82620f), so the field order mirrors the
// v95 PDB-clean send MINUS the v95-only repeatSkillPoint; the envelope's PRESENCE
// is proven by DR_check. The JMS gate-fix corrected the original GMS-only gates
// (which had JMS185 decoding the lean v83 header and misaligning every per-target
// field) AND a later over-correction that wrongly added the v95 repeatSkillPoint.
// packet-audit:verify packet=summon/serverbound/SummonAttackHandle version=jms_v185 ida=0x824a81
func TestSummonAttackDecodeJMS185(t *testing.T) {
	body := envelopeHeader(1000005, false)
	body = append(body, mobBlock(2000001, 9300018, 100, 1234)...)
	body = append(body, mobBlock(2000002, 9300166, -50, 5678)...)
	body = append(body, le32(0xABCD)...) // skillCRC

	ctx := test.CreateContext("JMS", 185, 1)
	l, _ := testlog.NewNullLogger()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m Attack
	m.Decode(l, ctx)(&reader, nil)

	assertAttack(t, &m, &reader, 1000005)
}

// envelopeHeader builds the v87/v95 anti-hack envelope header for 2 targets.
// withRepeatSkillPoint appends the v95-only trailing int after the positions.
func envelopeHeader(summonId uint32, withRepeatSkillPoint bool) []byte {
	b := []byte{}
	b = append(b, le32(summonId)...)
	b = append(b, le32(0x11111111)...) // ~drInfo[0]
	b = append(b, le32(0x22222222)...) // ~drInfo[1]
	b = append(b, le32(123456)...)     // updateTime
	b = append(b, le32(0x33333333)...) // ~drInfo[2]
	b = append(b, le32(0x44444444)...) // ~drInfo[3]
	b = append(b, 0x83)                // action|left
	b = append(b, le32(0xDEADBEEF)...) // dwKey
	b = append(b, le32(0xCAFEBABE)...) // crc32
	b = append(b, 0x02)                // count = 2
	b = append(b, le16(500)...)        // userX
	b = append(b, le16(600)...)        // userY
	b = append(b, le16(510)...)        // summonX
	b = append(b, le16(590)...)        // summonY
	if withRepeatSkillPoint {
		b = append(b, le32(75)...) // repeatSkillPoint (v95 only)
	}
	return b
}

func assertAttack(t *testing.T, m *Attack, reader *request.Reader, wantSummonId uint32) {
	t.Helper()
	if m.SummonId() != wantSummonId {
		t.Errorf("summonId = %d, want %d", m.SummonId(), wantSummonId)
	}
	if m.Direction() != 0x83 {
		t.Errorf("direction = %#x, want 0x83", m.Direction())
	}
	if len(m.Targets()) != 2 {
		t.Fatalf("targets len = %d, want 2", len(m.Targets()))
	}
	t0 := m.Targets()[0]
	if t0.MonsterOid() != 2000001 || t0.TemplateId() != 9300018 || t0.Damage() != 1234 || t0.Delay() != 100 {
		t.Errorf("target[0] = %+v, want oid=2000001 tmpl=9300018 dmg=1234 delay=100", t0)
	}
	t1 := m.Targets()[1]
	if t1.MonsterOid() != 2000002 || t1.TemplateId() != 9300166 || t1.Damage() != 5678 || t1.Delay() != -50 {
		t.Errorf("target[1] = %+v, want oid=2000002 tmpl=9300166 dmg=5678 delay=-50", t1)
	}
	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

func TestSummonAttackRoundTrip(t *testing.T) {
	in := Attack{
		summonId:  2000001,
		direction: 3,
		targets: []AttackTarget{
			{monsterOid: 1000001, templateId: 9300018, damage: 1234, delay: 100},
			{monsterOid: 1000002, templateId: 9300166, damage: 5678, delay: -50},
		},
	}

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out Attack
			out.Decode(l, ctx)(&reader, nil)

			if out.SummonId() != in.summonId {
				t.Errorf("summonId = %d, want %d", out.SummonId(), in.summonId)
			}
			if out.Direction() != in.direction {
				t.Errorf("direction = %d, want %d", out.Direction(), in.direction)
			}
			if len(out.Targets()) != len(in.targets) {
				t.Fatalf("targets len = %d, want %d", len(out.Targets()), len(in.targets))
			}
			for i := range in.targets {
				if out.Targets()[i].MonsterOid() != in.targets[i].monsterOid {
					t.Errorf("target[%d] monsterOid = %d, want %d", i, out.Targets()[i].MonsterOid(), in.targets[i].monsterOid)
				}
				if out.Targets()[i].Damage() != in.targets[i].damage {
					t.Errorf("target[%d] damage = %d, want %d", i, out.Targets()[i].Damage(), in.targets[i].damage)
				}
				if out.Targets()[i].Delay() != in.targets[i].delay {
					t.Errorf("target[%d] delay = %d, want %d", i, out.Targets()[i].Delay(), in.targets[i].delay)
				}
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
			}
		})
	}
}
