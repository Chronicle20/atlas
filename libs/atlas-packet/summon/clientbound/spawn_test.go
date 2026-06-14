package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonSpawn(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0 /*MovementStationary*/, true /*puppet*/, false /*animated*/)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)

			// oid round-trips on ALL versions: the wire is cid, oid, skillId on
			// every version (live x32dbg confirmed on v83 — the active OnCreated's
			// dispatcher pre-reads cid, so the int after cid is the oid).
			l, _ := testlog.NewNullLogger()
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out SummonSpawn
			out.Decode(l, ctx)(&reader, nil)
			if out.OwnerId() != 42 {
				t.Errorf("ownerId = %d, want 42", out.OwnerId())
			}
			if out.Oid() != 1000001 {
				t.Errorf("oid = %d, want 1000001", out.Oid())
			}
			if out.SkillId() != 3111002 {
				t.Errorf("skillId = %d, want 3111002", out.SkillId())
			}
		})
	}
}

// summonSpawnV83Body is the classic (pre-95) layout. It carries the oid: the
// ACTIVE v83 spawn dispatch (field path → OnCreated @0x95ADEC) has the dispatcher
// pre-read cid, so OnCreated reads oid, skillId, charLevel, SLV — i.e. the wire is
// cid, oid, skillId (matches Cosmic spawnSummon). Live x32dbg confirmed: at
// OnCreated's first Decode4 the read offset is already past cid, and omitting the
// oid makes the client read skillId into the cid slot and close at the foothold
// Decode2. (The earlier "no oid" reading analyzed the INACTIVE OnCreated @0x938F61,
// whose dispatcher does not pre-read cid — wrong path. See summon-wire-truth.md.)
//
//	int ownerId=42, int oid=1000001=0x000F4241, int skillId=3111002=0x002F785A,
//	byte 0x0A (charLevel, visual-only), byte level=20, short x=100, short y=-50,
//	byte stance=0, short 0 (foothold, visual-only), byte movementType=0,
//	bool !puppet=!true=0, bool !animated=!false=1.
var summonSpawnV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // ownerId
	0x41, 0x42, 0x0F, 0x00, // oid=1000001
	0x5A, 0x78, 0x2F, 0x00, // skillId
	0x0A,       // charLevel (visual-only)
	0x14,       // level=20
	0x64, 0x00, // x=100
	0xCE, 0xFF, // y=-50
	0x00,       // stance
	0x00, 0x00, // foothold (visual-only)
	0x00, // movementType
	0x00, // !puppet
	0x01, // !animated
}

// TestSummonSpawnBytesV83 pins the v83 layout: cid, oid, skillId, charLevel, SLV
// then the Init blob. The active v83 field dispatch is OnCreated @0x95ADEC, whose
// dispatcher pre-reads cid (so OnCreated reads oid, then skillId) — confirmed live
// in x32dbg (the prior markers below analyzed the INACTIVE OnCreated @0x938F61,
// whose dispatcher does NOT pre-read cid, hence the wrong "no oid" reading).
// NOTE: v84/v87/jms inherit this correction by the same dispatcher logic + Cosmic
// (spawnSummon always writes ownerId, objectId, skillId), but have NOT been
// re-confirmed live — their coverage-matrix cells need re-verification against the
// cid-pre-reading dispatcher (the old ida= markers below point at the wrong path).
func TestSummonSpawnBytesV83(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonSpawnV83Body)
	}
}

// TestSummonSpawnBytesV95 pins the v95 DELTA over the shared body: just a trailing
// bAvatarLook-present byte = 0 (the oid is now part of the shared body on all
// versions). For our 21-summon v83 roster no avatar look is carried and Tesla Coil
// is out of roster, so no AvatarLook blob / triangle tail follows.
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v95 ida=0x75a9a0
func TestSummonSpawnBytesV95(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// the shared body (cid, oid, skillId, ...), then bAvatarLook present = 0.
	want := append(append([]byte{}, summonSpawnV83Body...), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
	if len(got) != len(summonSpawnV83Body)+1 {
		t.Fatalf("v95 len = %d, want v83 len + 1 (avatarLook) = %d", len(got), len(summonSpawnV83Body)+1)
	}
}

// TestSummonSpawnBytesJMS185 pins the JMS185 spawn wire: the shared body (cid, oid,
// skillId, ...) PLUS a trailing bAvatarLook present-byte (jms185 Init reader
// sub_823AED@0x823aed: Decode1 bAvatarLook@0x823b99, then `if (v8) AvatarLook::
// Decode`@0x823bb0). The avatar-look tail is the JMS185 delta over GMS v83/v84/v87
// (the GMS-only avatar gate had JMS falling through 1 byte short — fixed earlier).
// NOTE: the oid is now written on all versions (see summon-wire-truth.md / the v83
// live-debugger finding); JMS185 inherits it by inference and has NOT been
// re-confirmed live — its matrix cell needs re-verification against the
// cid-pre-reading dispatcher (the old ida=0x9f80f8 marker analyzed the non-pre-read
// path).
func TestSummonSpawnBytesJMS185(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("JMS", 185, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// classic v83 body (no oid), then bAvatarLook present = 0.
	want := append(append([]byte{}, summonSpawnV83Body...), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("JMS185 bytes = % X, want % X", got, want)
	}
	if len(got) != len(summonSpawnV83Body)+1 {
		t.Fatalf("JMS185 len = %d, want v83 len + 1 (avatarLook) = %d", len(got), len(summonSpawnV83Body)+1)
	}
}
