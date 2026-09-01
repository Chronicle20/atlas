package character

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterDataNoByteMovement is the FR-9 regression guard: for a job an
// Atlas tenant can create today that is neither an Evan (22xx / 2001) nor a
// Dual Blade (43x), and whose skills are outside GMS v95's
// is_ignore_master_level_for_common list (@0x47cc20), the CharacterData wire
// bytes must be identical before and after task-275.
//
// The goldens below were captured from the encoder as it stood at the parent
// commit of task-275's first code change. If a later change to a version arm
// moves one of these bytes, that arm is reaching a character it must not.
func TestCharacterDataNoByteMovement(t *testing.T) {
	mk := func() CharacterData {
		return CharacterData{
			Stats: CharacterStats{
				Id: 1000, Name: "Ranger", Gender: 0, SkinColor: 1,
				Face: 20000, Hair: 30000,
				Level: 120, JobId: 312, Str: 100, Dex: 250, Int: 30, Luk: 20,
				Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
				Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
				GachaExp: 0, MapId: 100000000, SpawnPoint: 0,
			},
			BuddyCapacity: 20,
			Meso:          100000,
			Inventory: InventoryData{
				EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
				EtcCapacity: 24, CashCapacity: 24,
				EquipSlotExtExpire: 94354848000000000,
			},
			Skills: []SkillEntry{
				// job 311 (3rd job) — no master level on any version.
				{Id: 3110000, Level: 20, Expiration: -1},
				// job 312 (4th job) — master level on every version; NOT one of
				// the sixteen v95 ignore-list ids (312's are 3120010/3120011).
				{Id: 3121002, Level: 30, Expiration: -1, MasterLevel: 30},
			},
		}
	}

	for _, c := range []struct {
		name    string
		region  string
		major   uint16
		wantHex string
	}{
		{"GMS v84", "GMS", 84, "ffffffffffffffff00e803000052616e676572000000000000000001204e0000307500000000000000000000000000000000000000000000000000007838016400fa001e00140088138813b80bb80b0500030050c300000a000000000000e1f50500000000001400a086010018181818180040e0fd3b374f01000000000000000000000000020070742f0014000000ffffffffffffffff6a9f2f001e000000ffffffffffffffff1e0000000000000000000000000000000000ffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3b00000000000000000000000000"},
		{"GMS v95", "GMS", 95, "ffffffffffffffff00e803000052616e676572000000000000000001204e0000307500000000000000000000000000000000000000000000000000007838016400fa001e00140088138813b80bb80b0500030050c300000a000000000000e1f505000000000000001400a086010018181818180040e0fd3b374f01000000000000000000000000020070742f0014000000ffffffffffffffff6a9f2f001e000000ffffffffffffffff1e0000000000000000000000000000000000ffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3bffc99a3b000000000000"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cd := mk()
			got := hex.EncodeToString(pt.Encode(t, pt.CreateContext(c.region, c.major, 1), cd.Encode, nil))
			if got != c.wantHex {
				t.Errorf("%s CharacterData golden mismatch:\n got %s\nwant %s", c.name, got, c.wantHex)
			}
		})
	}
}
