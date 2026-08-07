package buff

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Model struct {
	sourceId  int32
	expiresAt time.Time
}

func NewModel(sourceId int32, expiresAt time.Time) Model {
	return Model{sourceId: sourceId, expiresAt: expiresAt}
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Expired() bool {
	return time.Now().After(m.expiresAt)
}

// HasActiveGmHide reports whether bs contains an unexpired SuperGmHide
// buff. bs's SourceId is a version-specific WIRE skill id (5101004 at
// v0.48, 9101004 at v0.62+), so it is resolved to its Identity through
// ctx's tenant version set before comparing (task-187) -- a raw compare
// against the canonical SuperGmHideId wire value would silently never
// match a v0.48 hide buff.
//
// Keying on the resolved identity, not the DARK_SIGHT stat type, so Rogue
// Dark Sight never matches.
func HasActiveGmHide(ctx context.Context, bs []Model) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		if id, ok := set.Skill.Resolve(skill.Id(b.SourceId())); ok && id == skill.SuperGmHide {
			return true
		}
	}
	return false
}
