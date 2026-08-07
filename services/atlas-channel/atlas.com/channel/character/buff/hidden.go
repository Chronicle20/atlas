package buff

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// IsGmHidden reports whether any buff in bs is an active GM-hide buff — one
// sourced from the SuperGM Hide skill and not yet expired. bs's SourceId is a
// version-specific WIRE skill id (5101004 at v0.48, 9101004 at v0.62+), so it
// is resolved to its Identity through ctx's tenant version set before
// comparing (task-187) -- a raw compare against the canonical SuperGmHideId
// wire value would silently never match a v0.48 hide buff.
//
// Keying on SourceId, NOT the DARK_SIGHT stat, is essential: Rogue Dark Sight
// also produces a DARK_SIGHT stat but must remain visible to other players.
// Only a SuperGmHide-sourced buff means "GM-hidden."
func IsGmHidden(ctx context.Context, bs []Model) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		if id, ok := set.Skill.Resolve(skill2.Id(b.SourceId())); ok && id == skill2.SuperGmHide {
			return true
		}
	}
	return false
}
