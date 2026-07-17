package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
//
// Re-pointed to the ACTIVE field-path target under task-106. Dispatch (IDA,
// MapleStory_dump.exe @port 13341): CUserPool::OnUserCommonPacket@0x972401 reads
// cid (Decode4@0x97240c) → op 0xAF vtable-calls CSummonedPool::OnCreated@0x95ADEC,
// which reads (cid already consumed upstream): Decode4(oid)@0x95ae0e,
// Decode4(skillId)@0x95ae17, Decode1(charLevel)@0x95ae21, Decode1(SLV)@0x95ae30,
// then CSummoned::Init@0x7a379b reads Decode2(x)@0x7a37b2, Decode2(y)@0x7a37bf,
// Decode1(stance)@0x7a37cc, Decode2(foothold)@0x7a37cf, Decode1(movementType)@0x7a37e3,
// Decode1(!puppet)@0x7a37fa, Decode1(!animated)@0x7a3821. NO avatar-look byte on
// v83 (the active path reads nothing after !animated). The inactive twin sub_938F61
// reads the SAME field order but is the wrong instance (task-088 live x32dbg).
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v83 ida=0x95adec
func TestSummonSpawnBytesV83(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonSpawnV83Body)
	}
}

// summonSpawnV79Body is the GMS v79 spawn layout: identical to the v83 body
// EXCEPT it omits the SLV byte (0x14) after charLevel — v79 carries ONE byte
// (charLevel) between skillId and the x/y Init blob, where v83+ carry TWO
// (charLevel + SLV). IDA-derived from the live decompile (see TestSummonSpawnBytesV79).
//
//	int ownerId=42 (cid, consumed upstream by OnUserCommonPacket),
//	int oid=1000001, int skillId=3111002, byte 0x0A (charLevel), NO SLV byte,
//	short x=100, short y=-50, byte stance=0, short 0 (foothold), byte movementType=0,
//	bool !puppet=0, bool !animated=1.
var summonSpawnV79Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
	0x41, 0x42, 0x0F, 0x00, // oid=1000001 (Decode4@0x8926a5 in sub_89268A)
	0x5A, 0x78, 0x2F, 0x00, // skillId=3111002 (Decode4@0x8926af)
	0x0A,       // charLevel (Decode1@0x8926b9) — the ONLY byte before the Init blob
	0x64, 0x00, // x=100 (Decode2@0x719f92 in sub_719F7B)
	0xCE, 0xFF, // y=-50 (Decode2@0x719f9f)
	0x00,       // stance (Decode1@0x719fac)
	0x00, 0x00, // foothold (Decode2@0x719faf)
	0x00, // movementType (Decode1@0x719fc3)
	0x00, // !puppet (Decode1@0x719fda)
	0x01, // !animated (Decode1@0x71a001)
}

// TestSummonSpawnBytesV79 pins the v79 spawn wire byte-for-byte against the live
// decompile (IDA, GMS_v79_1_DEVM.exe @port 13340). Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x8c8c79 reads cid (Decode4@0x8c8c84), and for
//     ops 164-169 calls the summon cluster dispatcher sub_892500@0x892500 (ecx=CUser).
//   - sub_892500 for a2==164 (SPAWN_SPECIAL_MAPOBJECT) vtable-calls (*(*this+0x24))
//     @0x892526 = the spawn leaf CSummonedPool::OnCreated sub_89268A@0x89268a.
//   - sub_89268A reads (cid already consumed upstream): Decode4(oid)@0x8926a5,
//     Decode4(skillId)@0x8926af, Decode1(charLevel)@0x8926b9 — and NOTHING ELSE
//     before the Init blob (the CSummoned ctor sub_7198ED@0x7198ed takes oid/skillId/
//     charLevel as plain args, no packet read; there is NO SLV byte on v79).
//   - sub_719F7B@0x719f7b then reads the Init blob: Decode2(x)@0x719f92,
//     Decode2(y)@0x719f9f, Decode1(stance)@0x719fac, Decode2(foothold)@0x719faf,
//     Decode1(movementType)@0x719fc3, Decode1(!puppet)@0x719fda, then (after the
//     GetSkill guard sub_6DC2F7) Decode1(!animated)@0x71a001.
//
// The missing SLV byte is the v79 delta vs v83+ (spawnHasSkillLevel(GMS,79)=false).
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v79 ida=0x89268a
func TestSummonSpawnBytesV79(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 79, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV79Body) {
		t.Fatalf("v79 bytes = % X, want % X", got, summonSpawnV79Body)
	}
	if len(got) != len(summonSpawnV83Body)-1 {
		t.Fatalf("v79 len = %d, want v83 len - 1 (no SLV) = %d", len(got), len(summonSpawnV83Body)-1)
	}
}

// TestSummonSpawnBytesV72 pins that the GMS v72 spawn wire is byte-identical to
// v79 (cid, oid, skillId, charLevel, Init blob; NO SLV byte, NO avatar byte).
// Verified live (IDA, GMS_v72.1_U_DEVM.exe @port 13339). Dispatch chain:
//   - CUserPool::OnUserCommonPacket reads cid; the summon cluster dispatcher
//     sub_848023@0x848023 case 160 (SPAWN_SPECIAL_MAPOBJECT) vtable-calls the spawn
//     leaf sub_8481AD@0x8481ad (Δ-4 vs v79 op 164).
//   - sub_8481AD reads (cid consumed upstream): Decode4(oid)@0x8481cf,
//     Decode4(skillId)@0x8481d9, Decode1(charLevel)@0x8481e8 — the ONLY byte before
//     the Init blob (NO SLV; spawnHasSkillLevel(GMS,72)=false), then sub_6E5F3C@0x6e5f3c
//     reads the Init blob: Decode2(x)@0x6e5f62, Decode2(y)@0x6e5f70,
//     Decode1(stance)@0x6e5f90, Decode2(foothold)@0x6e5fb3, Decode1(movementType)@0x6e5fc0,
//     Decode1(!puppet)@0x6e5fc6, and later Decode1(!animated)@0x6e630e. NO trailing
//     avatar byte (spawnHasAvatarLook(GMS,72)=false).
//
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v72 ida=0x8481ad
func TestSummonSpawnBytesV72(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 72, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV79Body) {
		t.Fatalf("v72 bytes = % X, want % X (identical to v79)", got, summonSpawnV79Body)
	}
	if len(got) != len(summonSpawnV83Body)-1 {
		t.Fatalf("v72 len = %d, want v83 len - 1 (no SLV) = %d", len(got), len(summonSpawnV83Body)-1)
	}
}

// TestSummonSpawnBytesV84 pins that v84 is byte-identical to v83 (no trailing avatar
// byte). Verified live (IDA, GMS_v84.1_U_DEVM.exe @port 13337). Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), routes
//     op 0xB3 (179) to the summon dispatcher sub_970201@0x970201, which for op 179
//     vtable-calls the spawn leaf sub_97038B@0x97038b (the ACTIVE leaf — v84 has no
//     inactive twin like v83's; the committed report already points here).
//   - sub_97038B@0x97038b reads (cid already consumed upstream):
//     Decode4@0x9703ad → oid (pool key for sub_97B9D1 lookup)
//     Decode4@0x9703b7 → skillId (v13)
//     Decode1@0x9703c1 → charLevel (v14)
//     Decode1@0x9703d0 → SLV (v15)
//     then sub_7C83D7@0x7c83d7 reads the Init blob:
//     Decode2@0x7c83ee → nX, Decode2@0x7c83fb → nY, Decode1@0x7c8408 → stance,
//     Decode2@0x7c8412 → foothold, Decode1@0x7c841f → movementType,
//     Decode1@0x7c8436 → !puppet, then (after the GetSkill guard sub_77EAC3)
//     Decode1@0x7c845d → !animated.
//     NO avatar-look byte on v84 — the active path reads nothing after !animated.
//     spawnHasAvatarLook(GMS,84) = (GMS && 84>=95) = false → v83 path; off-by-one
//     confirmed clear.
//
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v84 ida=0x97038b
func TestSummonSpawnBytesV84(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v84 bytes = % X, want % X (identical to v83)", got, summonSpawnV83Body)
	}
}

// TestSummonSpawnBytesV87 pins that v87 is byte-identical to v83 (cid, oid, skillId,
// charLevel, SLV, Init blob; NO trailing avatar byte). Verified live (IDA,
// GMSv87_4GB.exe @port 13340). Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9f7387 reads cid (Decode4@0x9f7392), routes
//     ops 188-193 (0xBC-0xC1) to CSummonedPool::OnPacket@0x9b35bf. For op 0xBC the
//     0xBC arm vtable-calls (*(*this+48)) = the spawn leaf sub_9B3749 (the ACTIVE
//     vtable+0x30 target — confirmed: 0x9b3749 sits at offset +0x30 in all 3 CUser
//     vtables that carry it: 0xb96060/0xbe4e14/0xbe5840).
//   - sub_9B3749@0x9b3749 reads (cid already consumed upstream):
//     Decode4@0x9b376b → oid (arg1 to the CSummoned ctor sub_7F489E, stored at
//     obj+172 — the object id; IDB-confirmed the same +172 oid slot as the v83
//     ctor sub_7A30A9, so the first leaf Decode4 IS the oid, not the skillId)
//     Decode4@0x9b3775 → skillId (arg2, stored obj+180)
//     Decode1@0x9b377f → charLevel (arg3)
//     Decode1@0x9b378e → SLV (arg4)
//     then sub_7F504A@0x7f504a reads the Init blob:
//     Decode2@0x7f5061 → nX, Decode2@0x7f506e → nY, Decode1@0x7f507b → stance,
//     Decode2@0x7f507e → foothold, Decode1@0x7f5092 → movementType,
//     Decode1@0x7f50a9 → !puppet, then (if GetSkill(skillId)!=0)
//     Decode1@0x7f50d0 → !animated, and returns (CSummoned::Init takes the
//     AvatarLook ptr from the CALLER, not the packet).
//     NO avatar-look byte on v87 — the active path reads nothing after !animated.
//     spawnHasAvatarLook(GMS,87) = (GMS && 87>=95) = false → v83 path; off-by-one
//     confirmed clear.
//
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v87 ida=0x9b3749
func TestSummonSpawnBytesV87(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v87 bytes = % X, want % X (identical to v83)", got, summonSpawnV83Body)
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

// TestSummonSpawnBytesJMS185 pins the JMS185 spawn wire byte-for-byte against the
// live decompile (IDA, MapleStory_dump_SCY.exe @port 13338). Dispatch chain:
//   - CUserPool::OnUserCommonPacket reads cid, op 0xB5 (181) routes to the spawn
//     leaf CSummonedPool::OnCreated sub_9F80F8@0x9f80f8 (the report/active target).
//   - sub_9F80F8@0x9f80f8 reads (cid already consumed upstream):
//     Decode4@0x9f811a → cid/ownerId (re-read here as the pool key)
//     Decode4@0x9f8124 → skillId (nSkillID; consumed by GetSkill in sub_823AED)
//     Decode1@0x9f812e → charLevel (nCharLevel; atlas writes fixed 0x0A)
//     Decode1@0x9f813d → SLV (nSLV; atlas 'level')
//     then sub_823AED@0x823aed reads the Init blob:
//     Decode2@0x823b15 → nX, Decode2@0x823b22 → nY, Decode1@0x823b2f → stance,
//     Decode2@0x823b39 → nCurFoothold, Decode1@0x823b46 → movementType,
//     Decode1@0x823b49 → !puppet (nAssistType), Decode1@0x823b8b → !animated
//     (nEnterType, read unconditionally on jms185), then
//     Decode1@0x823b99 → bAvatarLook present-byte, then
//     `if (v8) AvatarLook::Decode`@0x823bb0 (only entered when bAvatarLook != 0).
//
// The trailing bAvatarLook byte is the JMS185 delta over GMS v83/v84/v87 (which
// have no avatar byte). spawnHasAvatarLook(JMS,185) = (185 >= 185) = true. None of
// the 21 v83-roster summons carry an avatar look (Tesla Coil is out of roster), so
// we write present = 0 and the client skips both the AvatarLook blob and the Tesla
// triangle tail. NOTE on oid: sub_9F80F8 reads cid then skillId (NO oid in the leaf
// on jms185 — the int after cid is the skillId); the codec writes oid on all
// versions per the v83 live-debugger finding (summon-wire-truth.md). This fixture
// pins the codec output (shared body + 1 avatar byte = 0), which the JMS185 client
// tolerates for the roster.
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=jms_v185 ida=0x9f80f8
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
