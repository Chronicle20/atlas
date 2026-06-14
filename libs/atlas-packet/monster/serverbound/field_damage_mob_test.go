package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v83 ida=0x6675a8
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v84 ida=0x67d4ea
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v87 ida=0x6a1c43
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v95 ida=0x654300
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=jms_v185 ida=0x6e3d2f
func TestFieldDamageMob(t *testing.T) {
	input := FieldDamageMob{mobCrc: 0xAABBCCDD, damage: 0x000003E7}

	// Golden bytes (v83 baseline). CMob::Update field-damage send @0x667d39:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	//   Encode4(nFieldDamage)          -> damage uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x667d39)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE = 999 (Encode4 @0x667d39)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FieldDamageMob layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
