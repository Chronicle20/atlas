package skill

// AranHiddenComboParent maps an Aran hidden combo-variant skill to the skill
// that grants it, reporting false for anything that is not such a variant.
//
// The four variants -- Full Swing and Over Swing at two and three swings --
// are never in a character's skill book: no SP is ever spent on them and they
// are excluded from SP reset (IsPointResetExcluded). The client nonetheless
// sends the variant id in the attack packet once the combo count escalates the
// swing, so a server-side gate keyed on skill ownership has to read the
// variant's level from its parent rather than reject the attack.
//
// version-stable per task-187 audit: the Aran 21xxxxxx branch does not remap
// across the provisioned GMS range (see tools/skill-job-id-guard.sh's
// version-scope note), so a raw Id compare is correct here.
func AranHiddenComboParent(id Id) (Id, bool) {
	switch id {
	case AranStage3FullSwingDoubleSwingId, AranStage3FullSwingTripleSwingId:
		return AranStage3FullSwingId, true
	case AranStage4OverswingDoubleSwingId, AranStage4OverswingTripleSwingId:
		return AranStage4OverSwingId, true
	}
	return 0, false
}
