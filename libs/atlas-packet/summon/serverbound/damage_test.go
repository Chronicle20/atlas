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
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v95 ida=0x74b730
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v83 ida=0x7a607a
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v87 ida=0x7f879a
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v84 ida=0x7cbaf6
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
