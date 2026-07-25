package buff

import (
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// IsGmHidden reports whether any buff in bs is an active GM-hide buff — one
// sourced from the SuperGM Hide skill (SuperGmHideId) and not yet expired.
//
// Keying on SourceId, NOT the DARK_SIGHT stat, is essential: Rogue Dark Sight
// (RogueDarkSightId) also produces a DARK_SIGHT stat but must remain visible
// to other players. Only a SuperGmHide-sourced buff means "GM-hidden."
func IsGmHidden(bs []Model) bool {
	for _, b := range bs {
		if b.SourceId() == int32(skill2.SuperGmHideId) && !b.Expired() {
			return true
		}
	}
	return false
}
