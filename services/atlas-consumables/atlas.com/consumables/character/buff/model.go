package buff

import (
	"atlas-consumables/character/buff/stat"
	"time"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

type Model struct {
	sourceId  int32
	level     byte
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
	noExpiry  bool
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Changes() []stat.Model {
	return m.changes
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) Expired() bool {
	if m.noExpiry {
		return false
	}
	return m.expiresAt.Before(time.Now())
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

func (m Model) NoExpiry() bool {
	return m.noExpiry
}

func NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) Model {
	return Model{
		sourceId:  sourceId,
		level:     level,
		duration:  duration,
		changes:   changes,
		createdAt: createdAt,
		expiresAt: expiresAt,
		noExpiry:  noExpiry,
	}
}

// IsZombified reports whether bs contains an unexpired buff carrying an
// UNDEAD stat change -- the ZOMBIFY disease. Slice-level because every
// caller already holds the drained list and the question is "does any of
// these". See task-256 FR-1.
func IsZombified(bs []Model) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.changes {
			if c.Type == charconst.TemporaryStatTypeUndead {
				return true
			}
		}
	}
	return false
}

// IsPotionLocked reports whether bs contains an unexpired buff carrying a
// STOP_PORTION stat change -- the Seal-style debuff that forbids potion use.
// Magnitude is never consulted: the WZ `x` value is a client-side display
// input, so presence is the entire predicate. Slice-level for the same
// reason IsZombified is: every caller already holds the drained list.
// See task-280 FR-3.
func IsPotionLocked(bs []Model) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.changes {
			if c.Type == charconst.TemporaryStatTypeStopPortion {
				return true
			}
		}
	}
	return false
}
