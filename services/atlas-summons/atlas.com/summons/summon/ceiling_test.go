package summon

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// TestFaithfulPhysicalParity asserts FaithfulMaxPerHit reproduces the
// physical ceiling formula exactly for a one-handed-sword wielder.
//
// Hand computation:
//
//	watk = max(totalWatk=200, 14) = 200
//	weapon = SWORD1H → multiplier 4.0, mainstat = str = 200, secondary = dex = 100
//	maxBaseDmg = ceil(((4.0*200 + 100)/100.0)*200)
//	           = ceil((900/100.0)*200) = ceil(9.0*200) = ceil(1800.0) = 1800
//	summonDmgMod = (1800 >= 438) ? 0.054f : 0.077f = 0.054f
//	maxDamage = (float) 1800 * (0.054f * 100) = 1800 * 5.4f = 9720.0
//	(int) 9720.0 = 9720
func TestFaithfulPhysicalParity(t *testing.T) {
	const (
		totalWatk uint32 = 200
		totalMatk uint32 = 0
		totalInt  uint32 = 0
		str       uint32 = 200
		dex       uint32 = 100
		luk       uint32 = 50
		effWatk   int16  = 100
		effMatk   int16  = 0
	)
	got := FaithfulMaxPerHit(false, totalWatk, totalMatk, totalInt, str, dex, luk, item.WeaponTypeOneHandedSword, effWatk, effMatk)
	const want int64 = 9720
	if got != want {
		t.Fatalf("physical FaithfulMaxPerHit = %d, want %d (hand-computed value)", got, want)
	}
}

// TestFaithfulMagicParity asserts the magic branch reproduces the
// INT-curve formula exactly.
//
// Hand computation:
//
//	matk = max(totalMatk=200, 14) = 200, totalInt = 120 (<= 1700)
//	maxbasedamage = 200 - 120 = 80
//	         += (int)(0.1996049769 * 120^1.290631341)
//	         120^1.290631341 = 482.454631..., * 0.1996049769 = 96.300346..., (int)=96
//	maxbasedamage = 80 + 96 = 176
//	base = (176 * 107) / 100 = 18832 / 100 = 188
//	maxDamage = 188 * (0.05 * effMatk=100) = 188 * 5.0 = 940.0
//	(int) 940.0 = 940
func TestFaithfulMagicParity(t *testing.T) {
	const (
		totalWatk uint32 = 0
		totalMatk uint32 = 200
		totalInt  uint32 = 120
		str       uint32 = 0
		dex       uint32 = 0
		luk       uint32 = 0
		effWatk   int16  = 0
		effMatk   int16  = 100
	)
	got := FaithfulMaxPerHit(true, totalWatk, totalMatk, totalInt, str, dex, luk, item.WeaponTypeNone, effWatk, effMatk)
	const want int64 = 940
	if got != want {
		t.Fatalf("magic FaithfulMaxPerHit = %d, want %d (hand-computed value)", got, want)
	}
}

// TestFaithfulBowParity exercises the bow/crossbow/gun branch (mainstat=dex,
// secondary=str, multiplier 3.4 for BOW) to prove weapon-type awareness.
//
// Hand computation:
//
//	watk = max(300, 14) = 300, weapon = BOW → mult 3.4, main = dex = 250, sec = str = 40
//	maxBaseDmg = ceil(((3.4*250 + 40)/100.0)*300)
//	           = ceil(((850 + 40)/100.0)*300) = ceil(8.9*300) = ceil(2670.0) = 2670
//	summonDmgMod = (2670 >= 438) ? 0.054f
//	maxDamage = 2670 * (0.054f * 80) = 2670 * 4.32f = 11534.4 → (int) = 11534
func TestFaithfulBowParity(t *testing.T) {
	got := FaithfulMaxPerHit(false, 300, 0, 0, 40, 250, 0, item.WeaponTypeBow, 80, 0)
	const want int64 = 11534
	if got != want {
		t.Fatalf("bow FaithfulMaxPerHit = %d, want %d (hand-computed value)", got, want)
	}
}

// TestFaithfulCeilingClamps confirms the clamp wiring still bounds excess and
// passes in-bound damage untouched.
func TestFaithfulCeilingClamps(t *testing.T) {
	max := FaithfulMaxPerHit(false, 200, 0, 0, 200, 100, 50, item.WeaponTypeOneHandedSword, 100, 0)
	if clampDamage(uint32(max)+5000, max) != uint32(max) {
		t.Fatalf("excess not clamped")
	}
	if clampDamage(uint32(max)-1, max) != uint32(max)-1 {
		t.Fatalf("in-bound damage altered")
	}
}

// TestClampNoCeiling: max <= 0 means "stats unavailable" → never clamp.
func TestClampNoCeiling(t *testing.T) {
	if clampDamage(123456, 0) != 123456 {
		t.Fatalf("max=0 must pass damage through unclamped")
	}
}
