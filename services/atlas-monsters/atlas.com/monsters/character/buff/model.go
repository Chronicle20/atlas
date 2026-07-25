package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
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
// buff. Keying on SourceId, not the DARK_SIGHT stat type, so Rogue Dark
// Sight never matches.
func HasActiveGmHide(bs []Model) bool {
	for _, b := range bs {
		if b.SourceId() == int32(skill.SuperGmHideId) && !b.Expired() {
			return true
		}
	}
	return false
}
