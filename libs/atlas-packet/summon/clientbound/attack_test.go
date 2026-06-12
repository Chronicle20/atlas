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

// TestSummonAttackBytes pins the exact wire layout. SummonAttack is
// byte-identical across all versions (summon-packet-delta.md §3.4), so a single
// v83 assertion guards against an accidental version branch.
func TestSummonAttackBytes(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=2000001=0x001E8481, byte 0 (char level), direction=3, count=2,
	// then per target {monsterOid, byte 6, damage}:
	//   {1000001=0x000F4241, 6, 1234=0x000004D2}
	//   {1000002=0x000F4242, 6, 5678=0x0000162E}
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x81, 0x84, 0x1E, 0x00, // oid
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
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = % X, want % X", got, want)
	}
}
