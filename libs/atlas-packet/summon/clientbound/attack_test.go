package clientbound

import (
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
