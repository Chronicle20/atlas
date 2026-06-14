package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonSpawn(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0 /*MovementStationary*/, true /*puppet*/, false /*animated*/)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)

			// oid only round-trips on v95+ (gated); pre-95 wire carries no oid but
			// DOES carry skillId (the int after cid is the skillId — IDB-confirmed).
			l, _ := testlog.NewNullLogger()
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out SummonSpawn
			out.Decode(l, ctx)(&reader, nil)
			if out.OwnerId() != 42 {
				t.Errorf("ownerId = %d, want 42", out.OwnerId())
			}
			if out.SkillId() != 3111002 {
				t.Errorf("skillId = %d, want 3111002", out.SkillId())
			}
			te := tenant.MustFromContext(ctx)
			if te.IsRegion("GMS") && te.MajorAtLeast(95) {
				if out.Oid() != 1000001 {
					t.Errorf("oid = %d, want 1000001", out.Oid())
				}
			} else if out.Oid() != 0 {
				t.Errorf("pre-95 oid = %d, want 0 (no oid on wire)", out.Oid())
			}
		})
	}
}

// summonSpawnV83Body is the classic (pre-95) layout. NO oid: the v83 spawn reader
// (CSummonedPool OnCreated = sub_938F61) reads cid, then skillId, charLevel, SLV
// directly — the int after cid is the skillId (consumed by GetSkill@CSkillInfo),
// NOT an oid (IDB-confirmed, summon-wire-truth.md):
//
//	int ownerId=42, int skillId=3111002=0x002F785A,
//	byte 0x0A (charLevel, visual-only), byte level=20, short x=100, short y=-50,
//	byte stance=0, short 0 (foothold, visual-only), byte movementType=0,
//	bool !puppet=!true=0, bool !animated=!false=1.
var summonSpawnV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // ownerId
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

// TestSummonSpawnBytesV83 pins the classic (pre-95) layout. v83/v84/v87/JMS185
// all share this exact byte sequence (no oid). v87 spawn = the op 0xBC vtable+0x30
// target sub_9B3749, which reads cid + skillId + charLevel + SLV (only ONE int
// between cid and the two bytes → no oid) then the Init blob sub_7F504A —
// IDB-confirmed identical to v83.
// v84 spawn = the field op 0xB3 vtable+0x30 target sub_97038B@0x97038b, which
// reads cid + skillId + charLevel + SLV (only ONE int between cid and the two
// bytes → no oid) then the Init blob sub_7C83D7 — GMS_v84.1 IDB-confirmed
// byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v83 ida=0x938f61
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v87 ida=0x9b3749
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v84 ida=0x97038b
func TestSummonSpawnBytesV83(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonSpawnV83Body)
	}
}

// TestSummonSpawnBytesV95 pins the v95+ DELTA (gated >= 95, GMS only): the oid
// int after ownerId, plus a trailing bAvatarLook-present byte = 0. For our
// 21-summon v83 roster no avatar look is carried and Tesla Coil is out of
// roster, so no AvatarLook blob / triangle tail follows.
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v95 ida=0x75a9a0
func TestSummonSpawnBytesV95(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=0x000F4241, then the classic body (minus its leading
	// ownerId int), then bAvatarLook present = 0.
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId
		0x41, 0x42, 0x0F, 0x00, // oid (v95+ only)
		0x5A, 0x78, 0x2F, 0x00, // skillId
		0x0A,       // charLevel
		0x14,       // level
		0x64, 0x00, // x
		0xCE, 0xFF, // y
		0x00,       // stance
		0x00, 0x00, // foothold
		0x00, // movementType
		0x00, // !puppet
		0x01, // !animated
		0x00, // bAvatarLook present (v95+)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
	if len(got) != len(summonSpawnV83Body)+5 {
		t.Fatalf("v95 len = %d, want v83 len + 4 (oid) + 1 (avatarLook) = %d", len(got), len(summonSpawnV83Body)+5)
	}
}

// TestSummonSpawnBytesJMS185 pins the JMS185 spawn wire: NO oid (jms185 keeps the
// summon pool cid-keyed — OnCreated reader sub_9F80F8@0x9f80f8 reads cid, skillId,
// charLevel, SLV with only ONE int between cid and the two bytes), but the Init
// blob DOES end with a trailing bAvatarLook present-byte (jms185 Init reader
// sub_823AED@0x823aed: Decode1 bAvatarLook@0x823b99, then `if (v8) AvatarLook::
// Decode`@0x823bb0). So the JMS185 body is the classic v83 body PLUS one
// avatar-look byte = 0 (no AvatarLook blob, no Tesla tail for our roster). This
// is the spawn wire bug the JMS gate-fix corrected: the GMS-only avatar-look gate
// had JMS185 falling through to the lean v83 shape (1 byte short).
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
