package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
			// oid only round-trips on v95+ (gated); pre-95 wire carries no oid.
			te := tenant.MustFromContext(ctx)
			if te.IsRegion("GMS") && te.MajorAtLeast(95) {
				if out.Oid() != 1000001 {
					t.Errorf("oid = %d, want 1000001", out.Oid())
				}
			} else if out.Oid() != 0 {
				t.Errorf("pre-95 oid = %d, want 0 (no oid on wire)", out.Oid())
			}
			if out.NewStance() != 6 {
				t.Errorf("newStance = %d, want 6", out.NewStance())
			}
		})
	}
}

// TestSummonSkillBytes pins the classic (pre-95) wire: cid + a single stance
// byte. There is NO summonSkillId int and NO oid on v83/v87 — OnHit@0x7a6e5a
// reads one Decode1, masks 0x7F (IDB-confirmed, summon-wire-truth.md).
// v83 SKILL behavior lives at OnHit@0x7a6e5a (the LOWER of the swapped
// skill/damage opcodes); the export key CSummonedPool::OnSkill records this addr.
// v87 SKILL behavior lives at OnHit@0x7f963b (op 0xC1): one Decode1, &0x7F,
// SetAttackAction@0x7f9695 — same single-byte shape, no oid.
// v84 SKILL behavior lives at sub_7CC920@0x7cc920 (field op 0xB7): one Decode1,
// &0x7F, SetAttackAction sub_7CBAD3 — byte-identical single-byte shape, no oid
// (GMS_v84.1 IDB-confirmed).
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v83 ida=0x7a6e5a
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v87 ida=0x7f963b
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v84 ida=0x7cc920
func TestSummonSkillBytes(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, newStance=6 (no skillId, no oid)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x06, // newStance
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
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
