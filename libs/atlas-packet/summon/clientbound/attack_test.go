package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
			// oid only round-trips on v95+ (gated); pre-95 wire carries no oid.
			te := tenant.MustFromContext(ctx)
			if te.IsRegion("GMS") && te.MajorAtLeast(95) {
				if out.Oid() != in.oid {
					t.Errorf("oid = %d, want %d", out.Oid(), in.oid)
				}
			} else if out.Oid() != 0 {
				t.Errorf("pre-95 oid = %d, want 0 (no oid on wire)", out.Oid())
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

// summonAttackV83Body is the classic (pre-95) SummonAttack wire layout, shared
// by the v83 and v95 byte assertions. NO oid on v83/v87 (the summon pool is
// cid-keyed; oid is a v95+ addition — IDB-confirmed, summon-wire-truth.md):
//
//	cid=42, byte 0 (char level), direction=3, count=2,
//	then per target {monsterOid, byte 6, damage}:
//	  {1000001=0x000F4241, 6, 1234=0x000004D2}
//	  {1000002=0x000F4242, 6, 5678=0x0000162E}
var summonAttackV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid
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

// TestSummonAttackBytes pins the classic (pre-95) layout. v83/v84/v87 share this
// exact sequence with NO oid and NO trailing byte (v87 reader
// CSummonedPool::OnAttack@0x7f904c has no trailing Decode1; v84 reader
// CSummonedPool::OnAttack sub_7CC338@0x7cc338 reads charLevel+action+count+
// per{mobOid; if!=0: byte+damage} with no trailing byte — GMS_v84.1
// IDB-confirmed byte-identical to v83).
// jms185 (CSummonedPool::OnAttack@0x828707) reads charLevel@0x82878d +
// action@0x82879b + count@0x8287db + per{mobOid@0x82880c; if!=0: byte@0x82881a +
// damage@0x82882d} with NO trailing byte and NO oid — jms185 IDB-confirmed
// byte-identical to v83. The TestSummonAttackRoundTrip variant loop covers JMS.
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v83 ida=0x7a6882
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v87 ida=0x7f904c
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v84 ida=0x7cc338
// packet-audit:verify packet=summon/clientbound/SummonAttack version=jms_v185 ida=0x828707
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

// TestSummonAttackBytesV95 pins the v95+ DELTA (gated >= 95, GMS only): the oid
// int after cid, plus a single trailing flag byte = 0 after the target loop
// (v95 client reader CSummoned::OnAttack@0x753340's Decode1@0x7534e1).
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v95 ida=0x759860
func TestSummonAttackBytesV95(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=2000001=0x001E8481, byte 0, direction=3, count=2, targets..., trailing 0
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x81, 0x84, 0x1E, 0x00, // oid (v95+ only)
		0x00,                   // char level
		0x03,                   // direction
		0x02,                   // count
		0x41, 0x42, 0x0F, 0x00, // target0 monsterOid
		0x06,                   // byte 6
		0xD2, 0x04, 0x00, 0x00, // target0 damage
		0x42, 0x42, 0x0F, 0x00, // target1 monsterOid
		0x06,                   // byte 6
		0x2E, 0x16, 0x00, 0x00, // target1 damage
		0x00, // trailing flag (v95+)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}
