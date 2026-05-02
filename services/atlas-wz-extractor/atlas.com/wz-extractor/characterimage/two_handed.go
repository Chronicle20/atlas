package characterimage

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// ResolveStance applies the two-handed override: if any equipped weapon (slot
// -11) is two-handed *and* its WZ data ships stand2 sprites, the rendered
// stance becomes "stand2" regardless of the requested value. Returns the
// override flag for observability.
//
// Why the asset check: bow, crossbow, claw, knuckle, gun, and polearm are all
// two-handed for stat purposes but their Character.wz entries only ship
// stand1/walk1/alert/jump — there is no stand2 weapon canvas to draw. Forcing
// the body to stand2 in those cases produces a visibly wrong pose where the
// body is in a two-handed grip while the weapon is rendered at the stand1
// angle. Only true 2H melee (sword/axe/mace) actually has stand2 in WZ; for
// everything else we keep the requested stance.
func ResolveStance(assetsRoot, requested string, equipment map[int]int) (string, bool) {
	weaponId, ok := equipment[-11]
	if !ok {
		return requested, false
	}
	if !item.IsTwoHanded(item.Id(weaponId)) {
		return requested, false
	}
	if !weaponHasStand2(assetsRoot, weaponId) {
		return requested, false
	}
	if requested == "stand2" {
		return "stand2", false
	}
	return "stand2", true
}

// weaponHasStand2 reports whether the extracted weapon template has a
// stand2/0 directory on disk. False means the WZ source doesn't ship a
// two-handed weapon pose for this template.
func weaponHasStand2(assetsRoot string, weaponId int) bool {
	if assetsRoot == "" {
		// Tests that don't set up a filesystem fall through to "yes" so the
		// pre-asset-check ResolveStance behaviour is preserved.
		return true
	}
	dir := filepath.Join(assetsRoot, "character-parts", strconv.Itoa(weaponId), "stand2", "0")
	_, err := os.Stat(dir)
	return err == nil
}
