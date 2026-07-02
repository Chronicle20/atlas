package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestDamageRoundTrip(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// summonDamageMobBody is the real client SEND (CSummoned::SetDamaged, mob-present
// branch) — identical across v83/v87/v95: summonId + attackIdx + damage +
// monsterTemplateId + dir<0 flag.
//
//	summonId=1000001=0x000F4241, attackIdx=2, damage=1234=0x000004D2,
//	monsterIdFrom(template)=9300018=0x008DE832, dir flag=0.
var summonDamageMobBody = []byte{
	0x41, 0x42, 0x0F, 0x00, // summonId
	0x02,                   // attackIdx
	0xD2, 0x04, 0x00, 0x00, // damage
	0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom (mob template id)
	0x00, // dir<0 flag
}

// TestDamageDecodeMob decodes the real mob-present send and asserts the cursor
// ends clean across every version (the body shape is version-independent). v87
// SetDamaged@0x7f879a emits a byte-identical body to v83/v95 (summonId + attackIdx
// + damage + mobTemplateId + dir<0 byte; 0xFE no-mob sentinel branch present).
// v84 SetDamaged sub_7CBAF6@0x7cbaf6 (op 0xB6) emits the byte-identical body too
// (GMS_v84.1 IDB-confirmed: COutPacket(182)+Encode4 cid + mob-path{attackIdx,
// damage,templateId,dir<0} or no-mob{0xFE,damage}).
// jms185 SetDamaged@0x828032 (op 0xB4) emits the byte-identical body too (jms185
// IDB-confirmed: COutPacket(0xB4)@0x82827e + Encode4 cid@0x828293 + mob-path
// {attackIdx@0x8282ba, damage@0x8282c3, templateId@0x8282e0, dir<0@0x8282f0} or
// no-mob{0xFE@0x8282a4, damage@0x8282ad}). The decoder has no version gate.
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v95 ida=0x74b730
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v83 ida=0x7a607a
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v87 ida=0x7f879a
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v84 ida=0x7cbaf6
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=jms_v185 ida=0x828032
func TestDamageDecodeMob(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			req := request.Request(append([]byte{}, summonDamageMobBody...))
			reader := request.NewRequestReader(&req, 0)
			var m Damage
			m.Decode(l, ctx)(&reader, nil)

			if m.SummonId() != 1000001 {
				t.Errorf("summonId = %d, want 1000001", m.SummonId())
			}
			if m.AttackIdx() != 2 {
				t.Errorf("attackIdx = %d, want 2", m.AttackIdx())
			}
			if m.Damage() != 1234 {
				t.Errorf("damage = %d, want 1234", m.Damage())
			}
			if m.MonsterIdFrom() != 9300018 {
				t.Errorf("monsterIdFrom = %d, want 9300018", m.MonsterIdFrom())
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes", reader.Available())
			}
		})
	}
}

// summonDamageNoMobBody is the real client SEND (CSummoned::SetDamaged,
// no-source-mob branch): summonId + 0xFE sentinel + damage (no template, no dir).
//
//	summonId=1000001=0x000F4241, sentinel=0xFE, damage=1234=0x000004D2.
var summonDamageNoMobBody = []byte{
	0x41, 0x42, 0x0F, 0x00, // summonId
	0xFE,                   // 0xFE sentinel (no source mob)
	0xD2, 0x04, 0x00, 0x00, // damage
}

// TestDamageDecodeNoMob decodes the 0xFE no-source-mob branch and asserts the
// cursor ends clean (monsterIdFrom stays 0).
func TestDamageDecodeNoMob(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			req := request.Request(append([]byte{}, summonDamageNoMobBody...))
			reader := request.NewRequestReader(&req, 0)
			var m Damage
			m.Decode(l, ctx)(&reader, nil)

			if m.SummonId() != 1000001 {
				t.Errorf("summonId = %d, want 1000001", m.SummonId())
			}
			if m.AttackIdx() != 0xFE {
				t.Errorf("attackIdx = %d, want 0xFE", m.AttackIdx())
			}
			if m.Damage() != 1234 {
				t.Errorf("damage = %d, want 1234", m.Damage())
			}
			if m.MonsterIdFrom() != 0 {
				t.Errorf("monsterIdFrom = %d, want 0", m.MonsterIdFrom())
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes", reader.Available())
			}
		})
	}
}

// TestDamageBytesV72Mob pins the v72 client SEND (CSummoned::SetDamaged) byte-for-
// byte against the live decompile (IDA, GMS_v72.1_U_DEVM.exe @port 13339). The v72
// send sub_6E8A64@0x6e8a64 builds COutPacket(171)@0x6e8ca6 and emits a body
// byte-identical to v79/v83/v87/v95: Encode4(summonId)@0x6e8cbb = *(this+39); then
// the mob-present branch Encode1(attackIdx)@0x6e8ce2, Encode4(damage)@0x6e8ceb,
// Encode4(monsterTemplateId)@0x6e8d08, Encode1(a7<0 dir)@0x6e8d18; or the no-mob
// branch Encode1(0xFE)@0x6e8ccc, Encode4(damage)@0x6e8cd5. No version gate on the body.
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v72 ida=0x6e8a64
func TestDamageBytesV72Mob(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 72, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x41, 0x42, 0x0F, 0x00, // summonId (Encode4@0x6e8cbb)
		0x00,                   // attackIdx (NewDamage default; Encode1@0x6e8ce2)
		0xD2, 0x04, 0x00, 0x00, // damage (Encode4@0x6e8ceb)
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom (Encode4@0x6e8d08)
		0x00, // dir<0 flag (Encode1@0x6e8d18)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 bytes = % X, want % X", got, want)
	}
}

// TestDamageBytesV61Mob pins the v61 client SEND (CSummoned::SetDamaged) byte-for-
// byte against the live decompile (IDA, GMS_v61.1_U_DEVM.exe @port 13338). The v61
// send sub_67BB40@0x67bb40 builds COutPacket(148)@0x67bd84 and emits a body
// byte-identical to v72/v79/v83..v95: Encode4(summonId)@0x67bd99 = *(this+38); then
// the mob-present branch Encode1(attackIdx)@0x67bdc0, Encode4(damage)@0x67bdc9,
// Encode4(monsterTemplateId)@0x67bde6, Encode1(a7<0 dir)@0x67bdf6; or the no-mob
// branch Encode1(0xFE)@0x67bdaa, Encode4(damage)@0x67bdb3. No version gate on the body.
// v72 op171 (Δ-23).
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v61 ida=0x67bb40
func TestDamageBytesV61Mob(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x41, 0x42, 0x0F, 0x00, // summonId (Encode4@0x67bd99)
		0x00,                   // attackIdx (NewDamage default; Encode1@0x67bdc0)
		0xD2, 0x04, 0x00, 0x00, // damage (Encode4@0x67bdc9)
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom (Encode4@0x67bde6)
		0x00, // dir<0 flag (Encode1@0x67bdf6)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 bytes = % X, want % X", got, want)
	}
}

// TestDamageBytesV79Mob pins the v79 client SEND (CSummoned::SetDamaged) byte-for-
// byte against the live decompile (IDA, GMS_v79_1_DEVM.exe @port 13340). The v79
// send sub_71C7A7@0x71c7a7 builds COutPacket(173)@0x71c9e9 and emits a body
// byte-identical to v83/v87/v95: Encode4(summonId)@0x71c9fe = *(this+42); then the
// mob-present branch Encode1(attackIdx)@0x71ca25, Encode4(damage)@0x71ca2e,
// Encode4(monsterTemplateId)@0x71ca4b, Encode1(dir<0)@0x71ca5b; or the no-mob branch
// Encode1(0xFE)@0x71ca0f, Encode4(damage)@0x71ca18. No version gate on the body.
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v79 ida=0x71c7a7
func TestDamageBytesV79Mob(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 79, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x41, 0x42, 0x0F, 0x00, // summonId (Encode4@0x71c9fe)
		0x00,                   // attackIdx (NewDamage default; Encode1@0x71ca25)
		0xD2, 0x04, 0x00, 0x00, // damage (Encode4@0x71ca2e)
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom (Encode4@0x71ca4b)
		0x00, // dir<0 flag (Encode1@0x71ca5b)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 bytes = % X, want % X", got, want)
	}
}

// TestDamageBytesMob pins the encoded mob-present body (NewDamage defaults
// attackIdx=0; the byte fixture below uses attackIdx=0 to match).
func TestDamageBytesMob(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x41, 0x42, 0x0F, 0x00, // summonId
		0x00,                   // attackIdx (NewDamage default)
		0xD2, 0x04, 0x00, 0x00, // damage
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
		0x00, // dir<0 flag
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = % X, want % X", got, want)
	}
}
