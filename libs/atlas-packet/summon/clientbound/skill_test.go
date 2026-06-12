package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonSkill(t *testing.T) {
	in := NewSummonSkill(42, 1320009, 6)
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
			if out.SummonSkillId() != 1320009 {
				t.Errorf("summonSkillId = %d, want 1320009", out.SummonSkillId())
			}
			if out.NewStance() != 6 {
				t.Errorf("newStance = %d, want 6", out.NewStance())
			}
		})
	}
}

// TestSummonSkillBytes pins the exact wire layout. SummonSkill is
// byte-identical across all versions (summon-packet-delta.md §3.6), so a single
// v83 assertion guards against an accidental version branch.
func TestSummonSkillBytes(t *testing.T) {
	in := NewSummonSkill(42, 1320009, 6)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, summonSkillId=1320009=0x00142449, newStance=6
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x49, 0x24, 0x14, 0x00, // summonSkillId
		0x06, // newStance
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = % X, want % X", got, want)
	}
}
