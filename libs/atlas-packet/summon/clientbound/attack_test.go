package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonAttackRoundTrip(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3 /*direction*/, targets)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out SummonAttack
			out.Decode(l, ctx)(&reader, nil)

			if out.CharacterId() != in.characterId {
				t.Errorf("characterId = %d, want %d", out.CharacterId(), in.characterId)
			}
			// oid round-trips on ALL versions: cid is read upstream by
			// CUserPool::OnUserCommonPacket, so the per-op Decode4 is the oid.
			if out.Oid() != in.oid {
				t.Errorf("oid = %d, want %d", out.Oid(), in.oid)
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
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
			}
		})
	}
}

// summonAttackV83Body is the v83 SummonAttack wire layout (also the v95 body
// minus its trailing flag byte). The cid is read upstream by CUserPool::
// OnUserCommonPacket@0x972401; CSummonedPool::OnPacket@0x938dd7 then does one
// Decode4 = the oid before OnAttack. (The prior "no oid" reading missed the
// upstream cid — see summon-wire-truth.md.)
//
//	cid=42, oid=2000001=0x001E8481, byte 0 (char level), direction=3, count=2,
//	then per target {monsterOid, byte 6, damage}:
//	  {1000001=0x000F4241, 6, 1234=0x000004D2}
//	  {1000002=0x000F4242, 6, 5678=0x0000162E}
var summonAttackV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid
	0x81, 0x84, 0x1E, 0x00, // oid=2000001
	0x00,                   // char level
	0x03,                   // direction
	0x02,                   // count
	0x41, 0x42, 0x0F, 0x00, // target0 monsterOid
	0x06,                   // byte 6
	0xD2, 0x04, 0x00, 0x00, // target0 damage
	0x42, 0x42, 0x0F, 0x00, // target1 monsterOid
	0x06,                   // byte 6
	0x2E, 0x16, 0x00, 0x00, // target1 damage
}

// TestSummonAttackBytes pins the v83 layout: cid + oid + body, NO trailing byte
// (the trailing flag is a genuine v95-only addition). The cid is read upstream by
// CUserPool::OnUserCommonPacket; CSummonedPool::OnPacket@0x938dd7 then does one
// Decode4 = the oid before OnAttack. (The prior "no oid" reading missed the
// upstream cid — see summon-wire-truth.md.) NOTE: v84/v87/jms inherit this
// correction; their matrix cells need re-verification against the cid-pre-reading
// dispatcher.
func TestSummonAttackBytes(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonAttackV83Body)
	}
}

// summonAttackV79Body is the GMS v79 attack layout: identical to the v83 body
// EXCEPT it omits the leading char-level byte (0x00) — v79 reads the action byte
// (direction) FIRST, where v83+ read charLevel then the action byte.
//
//	cid=42, oid=2000001=0x001E8481, NO char-level byte, direction=3, count=2,
//	then per target {monsterOid, byte 6, damage}:
//	  {1000001=0x000F4241, 6, 1234=0x000004D2}
//	  {1000002=0x000F4242, 6, 5678=0x0000162E}
var summonAttackV79Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
	0x81, 0x84, 0x1E, 0x00, // oid=2000001 (Decode4@0x89253f in sub_892500)
	0x03,                   // direction (Decode1@0x71d06f, v5&0x7F)
	0x02,                   // count (Decode1@0x71d08b)
	0x41, 0x42, 0x0F, 0x00, // target0 monsterOid (Decode4@0x71d0bf)
	0x06,                   // byte 6 (Decode1@0x71d0cd)
	0xD2, 0x04, 0x00, 0x00, // target0 damage (Decode4@0x71d0e0)
	0x42, 0x42, 0x0F, 0x00, // target1 monsterOid
	0x06,                   // byte 6
	0x2E, 0x16, 0x00, 0x00, // target1 damage
}

// TestSummonAttackBytesV79 pins the v79 attack wire byte-for-byte against the live
// decompile (IDA, GMS_v79_1_DEVM.exe @port 13340). Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x8c8c79 reads cid (Decode4@0x8c8c84), ops
//     164-169 → summon cluster sub_892500@0x892500; the else branch reads oid
//     (Decode4@0x89253f) then for a2==167 (SUMMON_ATTACK) calls the OnAttack leaf
//     sub_71CFE9@0x71cfe9.
//   - sub_71CFE9 reads, after the GetSkill guard (sub_6DC2F7): Decode1@0x71d06f →
//     action byte (v70=(b>>7)&1 bLeft, v66=b&0x7F direction) — NO leading charLevel
//     byte (v83+ read charLevel first); Decode1@0x71d08b → count (loop guard
//     `if (v6 > 0)`); per target: Decode4@0x71d0bf monsterOid; if(oid){ Decode1@
//     0x71d0cd byte(6); Decode4@0x71d0e0 damage } and NOTHING after the loop.
// The missing char-level byte is the v79 delta vs v83+ (t.MajorAtLeast(83)=false).
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v79 ida=0x71cfe9
func TestSummonAttackBytesV79(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 79, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV79Body) {
		t.Fatalf("v79 bytes = % X, want % X", got, summonAttackV79Body)
	}
	if len(got) != len(summonAttackV83Body)-1 {
		t.Fatalf("v79 len = %d, want v83 len - 1 (no charLevel) = %d", len(got), len(summonAttackV83Body)-1)
	}
}

// TestSummonAttackBytesV83 pins the v83 wire byte-for-byte against the live
// decompile. Dispatch chain (IDA, MapleStory_dump.exe @port 13341):
//   - CUserPool::OnUserCommonPacket@0x972401 reads cid (Decode4@0x97240c), routes
//     op 0xB2 to CSummonedPool::OnPacket@0x972490.
//   - CSummonedPool::OnPacket@0x938dd7 reads oid (Decode4@0x938e16), looks up the
//     summon, then case 0xB2 calls CSummonedPool::OnAttack(v9,v5,..)@0x938e9c.
//   - CSummonedPool::OnAttack@0x7a6882 reads, after the GetSkill guard:
//       Decode1@0x7a6908 → charLevel (stored *(this+184))
//       Decode1@0x7a6916 → action byte (v71=(b>>7)&1 left flag, v67=b&0x7F)
//       Decode1@0x7a6937 → count (loop guard `if (v9 > 0)`)
//       per target: Decode4@0x7a6966 monsterOid; if(oid){ Decode1@0x7a6974 byte(6);
//         Decode4@0x7a6987 damage }
//     and NOTHING after the loop — there is NO trailing byte on v83 (the trailing
//     flag is a v95-only addition, gated >=95 in the codec).
// Wire: int cid (upstream) + int oid + byte charLevel + byte direction + byte count
//       + per target {int monsterOid, byte 6, int damage}.
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v83 ida=0x7a6882
func TestSummonAttackBytesV83(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonAttackV83Body)
	}
}

// TestSummonAttackBytesV84 pins that v84 is byte-identical to v83 (cid + oid + body,
// NO trailing flag byte). Verified live (IDA, GMS_v84.1_U_DEVM.exe @port 13337):
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), routes
//     op 0xB6 (182) to the summon dispatcher sub_970201@0x970201.
//   - sub_970201@0x970201 reads oid (Decode4@0x970240), looks up the summon, then
//     case 182 calls the OnAttack leaf sub_7CC338@0x7cc338.
//   - sub_7CC338@0x7cc338 reads, after the GetSkill guard (sub_77EAC3):
//       Decode1@0x7cc3be → charLevel (stored *(a1+184))
//       Decode1@0x7cc3cc → action byte (v75=(b>>7)&1 left flag, v74=b&0x7F direction)
//       Decode1@0x7cc3ed → count (loop guard `if (v9 > 0)`)
//       per target: Decode4@0x7cc41c monsterOid; if(oid){ Decode1@0x7cc42a byte(6);
//         Decode4@0x7cc43d damage }
//     and NOTHING after the loop — NO trailing byte on v84. The trailing flag is a
//     v95-only addition (gated GMS && >=95 in the codec → false at v84, so this is
//     the v83 path; off-by-one confirmed clear).
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v84 ida=0x7cc338
func TestSummonAttackBytesV84(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV83Body) {
		t.Fatalf("v84 bytes = % X, want % X (identical to v83)", got, summonAttackV83Body)
	}
}

// TestSummonAttackBytesV87 pins that v87 is byte-identical to v83 (cid + oid + body,
// NO trailing flag byte). Verified live (IDA, GMSv87_4GB.exe @port 13340):
//   - CUserPool::OnUserCommonPacket@0x9f7387 reads cid (Decode4@0x9f7392), routes
//     ops 188-193 to CSummonedPool::OnPacket@0x9b35bf.
//   - CSummonedPool::OnPacket@0x9b35bf reads oid (Decode4@0x9b35fe), looks up the
//     summon, then case 0xBF calls the OnAttack leaf CSummonedPool::OnAttack@0x7f904c.
//   - CSummonedPool::OnAttack@0x7f904c reads, after the GetSkill guard:
//       Decode1@0x7f90d2 → charLevel (stored *(this+184))
//       Decode1@0x7f90e0 → action byte (v73=(b>>7)&1 left flag, v71=b&0x7F direction)
//       Decode1@0x7f9101 → count (loop guard `if (v8 > 0)`)
//       per target: Decode4@0x7f9130 monsterOid; if(oid){ Decode1@0x7f913e byte(6);
//         Decode4@0x7f9151 damage }
//     and NOTHING after the loop — NO trailing byte on v87. The trailing flag is a
//     v95-only addition (gated GMS && >=95 in the codec → false at v87, so this is
//     the v83 path; off-by-one confirmed clear).
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v87 ida=0x7f904c
func TestSummonAttackBytesV87(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV83Body) {
		t.Fatalf("v87 bytes = % X, want % X (identical to v83)", got, summonAttackV83Body)
	}
}

// TestSummonAttackBytesV95 pins the v95 DELTA over the shared body: a single
// trailing flag byte = 0 after the target loop (v95 client reader
// CSummoned::OnAttack@0x753340's Decode1@0x7534e1). The oid is now part of the
// shared body on all versions.
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v95 ida=0x759860
func TestSummonAttackBytesV95(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// the shared v83 body, then a trailing flag byte = 0 (v95+)
	want := append(append([]byte{}, summonAttackV83Body...), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}

// TestSummonAttackBytesJMS185 pins that jms185 is byte-identical to v83 (cid + oid +
// body, NO trailing flag byte). Verified live (IDA, MapleStory_dump_SCY.exe @port
// 13338):
//   - CUserPool::OnUserCommonPacket reads cid, op 0xB8 (184) routes to
//     CSummonedPool::OnPacket@0x9f7f6e, which reads oid (Decode4@0x9f7fad), looks up
//     the summon, then case 0xB8 calls the OnAttack leaf
//     CSummonedPool::OnAttack@0x828707.
//   - CSummonedPool::OnAttack@0x828707 reads, after the GetSkill guard:
//       Decode1@0x82878d → charLevel/first byte (v6, stored *(this+188))
//       Decode1@0x82879b → action byte (v9=b>>7 bLeft, v10=b&0x7F direction)
//       Decode1@0x8287db → count (v11; loop guard `if (v11 > 0)`)
//       per target: Decode4@0x82880c monsterOid; if(oid){ Decode1@0x82881a byte(6);
//         Decode4@0x82882d damage }
//     and NOTHING after the loop — NO trailing byte on jms185. The trailing flag is
//     a GMS>=95-only addition (codec gate `IsRegion("GMS") && MajorAtLeast(95)` →
//     false for JMS, so jms gets no trailing byte). The jms185 path is byte-identical
//     to v83.
// packet-audit:verify packet=summon/clientbound/SummonAttack version=jms_v185 ida=0x828707
func TestSummonAttackBytesJMS185(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("JMS", 185, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV83Body) {
		t.Fatalf("JMS185 bytes = % X, want % X (identical to v83, no trailing byte)", got, summonAttackV83Body)
	}
}
