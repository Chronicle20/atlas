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
