package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v83/v84/v87 inline this send into CMob::Update (no standalone UpdateTimeBomb
// function — see structures/applicability.md / RESUME-STATE.md); evidence is pinned
// only for v95/jms where the standalone function exists. The route still lands for
// all versions (wire shape identical).
// packet-audit:verify packet=monster/serverbound/MonsterMobTimeBombEnd version=gms_v95 ida=0x643c30
// packet-audit:verify packet=monster/serverbound/MonsterMobTimeBombEnd version=jms_v185 ida=0x6ef8f8
func TestMobTimeBombEnd(t *testing.T) {
	// Non-boss layout (v95 baseline). CMob::UpdateTimeBomb @0x643c30:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc; then localUser x,y (no boss block).
	nonBoss := MobTimeBombEnd{boss: false, mobCrc: 0xAABBCCDD, localX: 0x00000064, localY: 0x000000C8}
	gotNB := nonBoss.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	wantNB := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x643c30)
		0x64, 0x00, 0x00, 0x00, // localX uint32 LE = 100
		0xC8, 0x00, 0x00, 0x00, // localY uint32 LE = 200
	}
	if !bytes.Equal(gotNB, wantNB) {
		t.Fatalf("MobTimeBombEnd non-boss layout mismatch\n got % x\nwant % x", gotNB, wantNB)
	}

	// Boss layout: the bBoss branch inserts the body-rect x/y centre pair before
	// the local-user position.
	boss := MobTimeBombEnd{boss: true, mobCrc: 0xAABBCCDD, bossX: 0x00000010, bossY: 0x00000020, localX: 0x00000064, localY: 0x000000C8}
	gotB := boss.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	wantB := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD
		0x10, 0x00, 0x00, 0x00, // bossX uint32 LE = 16 (Encode4, bBoss branch)
		0x20, 0x00, 0x00, 0x00, // bossY uint32 LE = 32 (Encode4, bBoss branch)
		0x64, 0x00, 0x00, 0x00, // localX uint32 LE = 100
		0xC8, 0x00, 0x00, 0x00, // localY uint32 LE = 200
	}
	if !bytes.Equal(gotB, wantB) {
		t.Fatalf("MobTimeBombEnd boss layout mismatch\n got % x\nwant % x", gotB, wantB)
	}

	for _, v := range pt.Variants {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, nonBoss.Encode, nonBoss.Decode, nil)
			pt.RoundTrip(t, ctx, boss.Encode, boss.Decode, nil)
		})
	}
}
