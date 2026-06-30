package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonSkill(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)

			// Field-level assertions: encode then decode and verify fields.
			l, _ := testlog.NewNullLogger()
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out SummonSkill
			out.Decode(l, ctx)(&reader, nil)
			if out.CharacterId() != 42 {
				t.Errorf("characterId = %d, want 42", out.CharacterId())
			}
			// oid round-trips on ALL versions: cid is read upstream by
			// CUserPool::OnUserCommonPacket, so the per-op Decode4 is the oid.
			if out.Oid() != 1000001 {
				t.Errorf("oid = %d, want 1000001", out.Oid())
			}
			if out.NewStance() != 6 {
				t.Errorf("newStance = %d, want 6", out.NewStance())
			}
		})
	}
}

// TestSummonSkillBytes pins the v83 wire: cid + oid + a single stance byte. There
// is NO summonSkillId int (OnHit reads one Decode1, masks 0x7F). The cid is read
// upstream by CUserPool::OnUserCommonPacket@0x972401; CSummonedPool::OnPacket@
// 0x938dd7 then does one Decode4 = the oid before OnHit (the skill leaf, the LOWER
// of the swapped skill/damage opcodes). (The prior "no oid" reading missed the
// upstream cid — see summon-wire-truth.md.) NOTE: v84/v87/jms inherit this
// correction; their matrix cells need re-verification against the cid-pre-reading
// dispatcher.
func TestSummonSkillBytes(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (no skillId)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid
		0x06, // newStance
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesV79 pins the v79 wire byte-for-byte against the live
// decompile (IDA, GMS_v79_1_DEVM.exe @port 13340). v79 is v83-shaped (cid + oid +
// single stance byte). NOTE the v79 damage/skill opcode SWAP: the single-byte
// SKILL body is reached at op 0xA8 (168); the v79 seed template routes SummonSkill
// →0xA8 (matching the client) and SummonDamage→0xA9.
//   - CUserPool::OnUserCommonPacket@0x8c8c79 reads cid (Decode4@0x8c8c84), ops
//     164-169 → summon cluster sub_892500@0x892500; the else branch reads oid
//     (Decode4@0x89253f) then case 168 calls the skill leaf sub_71D5DF@0x71d5df.
//   - sub_71D5DF reads exactly ONE byte: Decode1@0x71d62e → sub_71C784(b & 0x7F)
//     — a single stance byte masked 0x7F, and nothing else. There is NO summonSkillId
//     int on the wire.
// Wire: int cid (upstream) + int oid + byte stance.
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v79 ida=0x71d5df
func TestSummonSkillBytesV79(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 79, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (single byte, masked 0x7F client-side)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x89253f in sub_892500)
		0x06, // newStance (Decode1@0x71d62e)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesV83 pins the v83 wire byte-for-byte against the live
// decompile. Dispatch chain (IDA, MapleStory_dump.exe @port 13341):
//   - CUserPool::OnUserCommonPacket@0x972401 reads cid (Decode4@0x97240c), routes
//     op 0xB4 to CSummonedPool::OnPacket@0x972490.
//   - CSummonedPool::OnPacket@0x938dd7 reads oid (Decode4@0x938e16), looks up the
//     summon, then case 0xB4 calls the skill leaf @0x938e86.
//   - The skill body lives at 0x7a6e5a (exported FName CSummonedPool::OnSkill; the
//     mangled symbol there is OnHit — the known naming swap; the body is what
//     matters). It reads exactly ONE byte: Decode1@0x7a6ea9 → sub_7A601D(this,
//     b & 0x7F) — a single stance byte masked 0x7F, and nothing else. There is NO
//     summonSkillId int on the wire in any version.
// Wire: int cid (upstream) + int oid + byte stance.
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v83 ida=0x7a6e5a
func TestSummonSkillBytesV83(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (single byte, masked 0x7F client-side)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x938e16 in OnPacket)
		0x06, // newStance (Decode1@0x7a6ea9)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesV84 pins the v84 wire byte-for-byte against the live
// decompile (IDA, GMS_v84.1_U_DEVM.exe @port 13337). v84 is v83-shaped — same
// dispatch + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), routes
//     op 0xB7 (183) to the summon dispatcher sub_970201@0x970201.
//   - sub_970201@0x970201 reads oid (Decode4@0x970240), looks up the summon, then
//     case 183 calls the skill leaf sub_7CC920@0x7cc920 (exported FName
//     CSummonedPool::OnSkill — the single-byte-stance body is what matters).
//   - sub_7CC920@0x7cc920 reads exactly ONE byte: Decode1@0x7cc96f → sub_7CBAD3(b &
//     0x7F) — a single stance byte masked 0x7F, and nothing else. There is NO
//     summonSkillId int on the wire in any version.
// Wire: int cid (upstream) + int oid + byte stance. Skill has no version gate, so the
// v84 path is byte-identical to v83 (off-by-one confirmed clear).
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v84 ida=0x7cc920
func TestSummonSkillBytesV84(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (single byte, masked 0x7F client-side)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x970240 in sub_970201)
		0x06, // newStance (Decode1@0x7cc96f)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesV87 pins the v87 wire byte-for-byte against the live
// decompile (IDA, GMSv87_4GB.exe @port 13340). v87 is v83-shaped — same dispatch
// + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9f7387 reads cid (Decode4@0x9f7392), routes
//     ops 188-193 to CSummonedPool::OnPacket@0x9b35bf.
//   - CSummonedPool::OnPacket@0x9b35bf reads oid (Decode4@0x9b35fe), looks up the
//     summon, then case 0xC1 calls the skill leaf @0x7f963b.
//   - The skill body lives at 0x7f963b (exported FName CSummonedPool::OnSkill; the
//     mangled symbol there is OnHit — the known naming swap; the body is truth).
//     It reads exactly ONE byte: Decode1@0x7f968a → SetAttackAction(this, v4 & 0x7F)
//     — a single stance byte masked 0x7F, and nothing else. There is NO summonSkillId
//     int on the wire in any version.
// Wire: int cid (upstream) + int oid + byte stance. Skill has no version gate, so the
// v87 path is byte-identical to v83 (off-by-one confirmed clear).
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v87 ida=0x7f963b
func TestSummonSkillBytesV87(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (single byte, masked 0x7F client-side)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x9b35fe in OnPacket)
		0x06, // newStance (Decode1@0x7f968a)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesV95 pins the v95+ DELTA: the oid int between cid and the
// stance byte. Still no summonSkillId int (v95 OnSkill also reads a single byte).
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v95 ida=0x759890
func TestSummonSkillBytesV95(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=0x000F4241, newStance=6
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid (v95+ only)
		0x06, // newStance
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}

// TestSummonSkillBytesJMS185 pins the JMS185 wire byte-for-byte against the live
// decompile (IDA, MapleStory_dump_SCY.exe @port 13338). jms185 is v83-shaped —
// same dispatch + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket reads cid, op 0xBA (186) routes to
//     CSummonedPool::OnPacket@0x9f7f6e, which reads oid (Decode4@0x9f7fad), looks up
//     the summon, then case 0xBA calls the skill leaf CSummonedPool::OnSkill@0x828cb2.
//   - The skill body lives at 0x828cb2 (exported FName CSummonedPool::OnSkill; the
//     mangled symbol there is OnHit — the known naming swap; the body is truth).
//     It reads exactly ONE byte: Decode1@0x828d01 → CSummoned::SetAttackAction(this,
//     v4 & 0x7F) — a single stance byte masked 0x7F, and nothing else. There is NO
//     summonSkillId int on the wire in any version.
// Wire: int cid (upstream) + int oid + byte stance. Skill has no version gate, so the
// jms185 path is byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonSkill version=jms_v185 ida=0x828cb2
func TestSummonSkillBytesJMS185(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("JMS", 185, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, newStance=6 (single byte, masked 0x7F client-side)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x9f7fad in OnPacket)
		0x06, // newStance (Decode1@0x828d01)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("JMS185 bytes = % X, want % X", got, want)
	}
}
