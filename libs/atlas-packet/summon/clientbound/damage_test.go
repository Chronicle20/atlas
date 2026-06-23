package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSummonDamage(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// summonDamageV83Body is the v83 wire: cid + oid + body, NO trailing dir byte.
// The cid is read upstream by CUserPool::OnUserCommonPacket@0x972401; CSummonedPool::
// OnPacket@0x938dd7 then does one Decode4 = the oid before the damage leaf
// (OnSkill@0x7a6ebe, the HIGHER swapped opcode), which reads attackIdx(b), dmg(i),
// if attackIdx>-2:{templateId(i), bLeft(b)} and nothing after. (The prior "no oid"
// reading missed the upstream cid — see summon-wire-truth.md.)
//
//	cid=42, oid=1000001=0x000F4241, attackIdx 12, damage=1234=0x000004D2,
//	monsterIdFrom=9300018=0x008DE832, bLeft 0
var summonDamageV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid
	0x41, 0x42, 0x0F, 0x00, // oid=1000001
	0x0C,                   // attackIdx (12)
	0xD2, 0x04, 0x00, 0x00, // damage
	0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
	0x00, // bLeft
}

// TestSummonDamageBytes pins the v83 layout: cid + oid + body, no trailing dir
// byte (the dir<0 byte belongs to the SERVERBOUND SetDamaged send, not this
// broadcast). (The prior "no oid" reading missed the upstream CUserPool cid read
// — see summon-wire-truth.md.) NOTE: v84/v87/jms inherit this correction; their
// matrix cells need re-verification against the cid-pre-reading dispatcher.
func TestSummonDamageBytes(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV83 pins the v83 wire byte-for-byte against the live
// decompile. Dispatch chain (IDA, MapleStory_dump.exe @port 13341):
//   - CUserPool::OnUserCommonPacket@0x972401 reads cid (Decode4@0x97240c), routes
//     op 0xB3 to CSummonedPool::OnPacket@0x972490.
//   - CSummonedPool::OnPacket@0x938dd7 reads oid (Decode4@0x938e16), looks up the
//     summon, then case 0xB3 calls the damage leaf @0x938e91.
//   - The damage body lives at 0x7a6ebe (exported FName CSummonedPool::OnHit; the
//     mangled symbol there is OnSkill — a known naming swap, the body is what
//     matters). It reads:
//       Decode1@0x7a6eef → attackIdx (v34=v6=b; atlas writes 12)
//       Decode4@0x7a6f04 → damage (v38)
//       if (attackIdx > -2):   // 12 > -2, branch fires
//         Decode4@0x7a6f16 → monsterIdFrom (v39 → GetMobTemplate)
//         Decode1@0x7a6f24 → bLeft (v35; atlas writes 0)
//     and nothing after — no trailing dir byte on any version (the dir<0 byte
//     belongs to the SERVERBOUND SetDamaged send).
// Wire: int cid (upstream) + int oid + byte attackIdx(12) + int damage +
//       int monsterIdFrom + byte bLeft(0).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v83 ida=0x7a6ebe
func TestSummonDamageBytesV83(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV87 pins that v87 is byte-identical to v83 (cid + oid +
// body, no trailing dir byte). Confirmed live (IDA, GMSv87_4GB.exe @port 13340):
//   - CUserPool::OnUserCommonPacket@0x9f7387 reads cid (Decode4@0x9f7392), routes
//     ops 188-193 to CSummonedPool::OnPacket@0x9b35bf.
//   - CSummonedPool::OnPacket@0x9b35bf reads oid (Decode4@0x9b35fe), looks up the
//     summon, then case 0xC0 calls the damage leaf @0x7f969f.
//   - The damage body lives at 0x7f969f (exported FName CSummonedPool::OnHit; the
//     mangled symbol there is OnSkill — the known naming swap, the body is truth).
//     It reads:
//       Decode1@0x7f96d0 → attackIdx (v4=v5; atlas writes 12)
//       Decode4@0x7f96e5 → damage (v37)
//       if (v5 > -2):   // 12 > -2, branch fires
//         Decode4@0x7f96f7 → monsterIdFrom (v38 → GetMobTemplate)
//         Decode1@0x7f9705 → bLeft (v34; atlas writes 0)
//     and nothing after from the packet — no trailing dir byte (the dir<0 byte
//     belongs to the SERVERBOUND SetDamaged send). Damage has no version gate, so
//     the v87 path is byte-identical to v83 (off-by-one confirmed clear).
// Wire: int cid (upstream) + int oid + byte attackIdx(12) + int damage +
//       int monsterIdFrom + byte bLeft(0).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v87 ida=0x7f969f
func TestSummonDamageBytesV87(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v87 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV84 pins that v84 is byte-identical to v83 (cid + oid + body,
// no trailing dir byte). Verified live (IDA, GMS_v84.1_U_DEVM.exe @port 13337):
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), routes
//     op 0xB8 (184) to the summon dispatcher sub_970201@0x970201.
//   - sub_970201@0x970201 reads oid (Decode4@0x970240), looks up the summon, then
//     case 184 calls the damage leaf sub_7CC984@0x7cc984 (exported FName
//     CSummonedPool::OnHit — the body that reads attackIdx/damage is what matters).
//   - sub_7CC984@0x7cc984 reads:
//       Decode1@0x7cc9b5 → attackIdx (v34=v5; atlas writes 12)
//       Decode4@0x7cc9ca → damage (v38)
//       if (v5 > -2):   // 12 > -2, branch fires
//         Decode4@0x7cc9dc → monsterIdFrom (v39 → sub_6938FA/GetMobTemplate)
//         Decode1@0x7cc9ea → bLeft (v35; atlas writes 0)
//     and nothing after from the packet — no trailing dir byte (the dir<0 byte
//     belongs to the SERVERBOUND SetDamaged send). Damage has no version gate, so
//     the v84 path is byte-identical to v83 (off-by-one confirmed clear).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v84 ida=0x7cc984
func TestSummonDamageBytesV84(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v84 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV95 pins that v95 is byte-identical to v83 for damage: the
// oid is now in the shared body and there is no v95-specific delta (v95 OnHit@
// 0x74bc80 stops at bLeft — the dir byte is serverbound only).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v95 ida=0x7598c0
func TestSummonDamageBytesV95(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v95 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}
