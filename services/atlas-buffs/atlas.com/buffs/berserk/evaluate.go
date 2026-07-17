package berserk

// Evaluate computes the berserk-active state (design D7 / PRD FR-1):
//
//	active := skillLevel > 0 && hp > 0 && hp*100/effectiveMaxHp < x
//
// Strict less-than is Cosmic parity (Character.java:1852): at exactly x% the
// aura is OFF. hp > 0 folds death handling into the formula — the
// death-accompanying STAT_CHANGED(HP=0) evaluates to inactive with no DIED
// consumer. effectiveMaxHp is buff-inclusive (atlas-effective-stats), so
// Hyper Body apply/expire can flip the state with hp constant. Integer math:
// hp is uint16 so hp*100 fits uint32 with no overflow.
func Evaluate(skillLevel byte, hp uint16, effectiveMaxHp uint32, x int16) bool {
	if skillLevel == 0 || hp == 0 || effectiveMaxHp == 0 || x <= 0 {
		return false
	}
	return uint32(hp)*100/effectiveMaxHp < uint32(x)
}
