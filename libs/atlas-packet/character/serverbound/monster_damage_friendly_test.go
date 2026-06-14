package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MonsterDamageFriendly is the serverbound MOB_DAMAGE_MOB_FRIENDLY packet, built
// by CMob::Update's friendly-damage send site (3xEncode4). task-092 Cluster-A
// promoted this pre-existing codec to verified.
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v83 ida=0x6675a8
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v84 ida=0x67d4ea
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v87 ida=0x6a1c43
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v95 ida=0x654300
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=jms_v185 ida=0x6e3d2f
func TestMonsterDamageFriendlyGolden(t *testing.T) {
	input := MonsterDamageFriendly{attackerId: 0x11223344, observerId: 0x0010F447, attackedId: 0xAABBCCDD}

	// Golden bytes (v83 baseline). CMob::Update friendly-damage send @0x667f50:
	//   Encode4(SecureFuse(this.m_dwMobID))     -> attackerId (the friendly/victim mob)
	//   Encode4(CWvsContext.dwCharacterID)      -> observerId (controlling character)
	//   Encode4(SecureFuse(attacker.m_dwMobID)) -> attackedId (the hostile attacker mob)
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerId uint32 LE = 0x11223344 (Encode4 @0x667f50)
		0x47, 0xF4, 0x10, 0x00, // observerId uint32 LE = 0x0010F447 (Encode4 @0x667f50)
		0xDD, 0xCC, 0xBB, 0xAA, // attackedId uint32 LE = 0xAABBCCDD (Encode4 @0x667f50)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterDamageFriendly layout mismatch\n got % x\nwant % x", got, want)
	}
}

func TestMonsterDamageFriendlyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := MonsterDamageFriendly{attackerId: 100, observerId: 200, attackedId: 300}
			output := MonsterDamageFriendly{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.AttackerId() != input.AttackerId() {
				t.Errorf("attackerId: got %v, want %v", output.AttackerId(), input.AttackerId())
			}
			if output.ObserverId() != input.ObserverId() {
				t.Errorf("observerId: got %v, want %v", output.ObserverId(), input.ObserverId())
			}
			if output.AttackedId() != input.AttackedId() {
				t.Errorf("attackedId: got %v, want %v", output.AttackedId(), input.AttackedId())
			}
		})
	}
}
