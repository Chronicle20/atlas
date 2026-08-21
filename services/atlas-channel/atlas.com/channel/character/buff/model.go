package buff

import (
	"atlas-channel/character/buff/stat"
	"time"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// IsMount reports whether this buff is a tamed/skill mount (carries a
// MONSTER_RIDING stat change). The mount is transient state: it is cancelled
// (not re-rendered) on login and auto-cancelled when the mount grows too tired.
func IsMount(m Model) bool {
	for _, c := range m.changes {
		if c.Type() == string(charconst.TemporaryStatTypeMonsterRiding) {
			return true
		}
	}
	return false
}

// IsZombified reports whether bs contains an unexpired buff carrying an
// UNDEAD stat change -- the ZOMBIFY disease. Slice-level because every caller
// already holds the drained list and the question is "does any of these".
// See task-256 FR-1.
func IsZombified(bs []Model) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.changes {
			if c.Type() == string(charconst.TemporaryStatTypeUndead) {
				return true
			}
		}
	}
	return false
}

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
