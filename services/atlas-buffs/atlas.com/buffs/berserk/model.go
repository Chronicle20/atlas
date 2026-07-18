package berserk

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Broadcast cadence and service-local pacing knobs. Exported so the character
// package's buff hook and tests reference the same values.
//
// BroadcastPeriod is the steady re-broadcast interval (refreshing late-joining
// map observers) once a schedule is running. There is intentionally no "initial
// delay": a state transition is emitted inline the moment the re-evaluation
// detects it (processor.go reevaluate), so the aura flips on the threshold
// crossing rather than a scan pass later. The former 5s initial delay was reset
// on every HP re-evaluation, so a stream of HP STAT_CHANGED events (sustained
// combat) pushed the deadline out indefinitely and the aura only appeared once
// HP stopped changing (task-154 live-test finding).
const (
	BroadcastPeriod = 3 * time.Second
	// ReevalGrace defers buff-origin re-evaluations so atlas-effective-stats can
	// consume the buff event and recompute max HP before we read it (design D5).
	ReevalGrace = 2 * time.Second
	// ReevalRetryDelay re-arms dirtyAt after a failed lookup so the re-evaluation
	// retries instead of silently freezing on stale state (design §4.1).
	ReevalRetryDelay = time.Second
)

// Model is one tracked Dark Knight. channelId is meaningless until
// channelKnown; entries created from skill UPDATED events (which carry no
// channel) stay unroutable until the next channel-bearing character event.
type Model struct {
	worldId         world.Id
	channelId       channel.Id
	channelKnown    bool
	characterId     uint32
	characterLevel  byte
	skillLevel      byte
	active          bool
	dirtyAt         time.Time
	nextBroadcastAt time.Time
}

func (m Model) WorldId() world.Id          { return m.worldId }
func (m Model) ChannelId() channel.Id      { return m.channelId }
func (m Model) ChannelKnown() bool         { return m.channelKnown }
func (m Model) CharacterId() uint32        { return m.characterId }
func (m Model) CharacterLevel() byte       { return m.characterLevel }
func (m Model) SkillLevel() byte           { return m.skillLevel }
func (m Model) Active() bool               { return m.active }
func (m Model) DirtyAt() time.Time         { return m.dirtyAt }
func (m Model) NextBroadcastAt() time.Time { return m.nextBroadcastAt }

// DirtyDue reports whether a re-evaluation is due. Requires channelKnown
// because the effective-stats route needs world/channel to resolve max HP.
func (m Model) DirtyDue(now time.Time) bool {
	return m.channelKnown && !m.dirtyAt.IsZero() && !m.dirtyAt.After(now)
}

// BroadcastDue reports whether a broadcast tick is due. Zero nextBroadcastAt
// means no evaluation has completed yet — nothing to broadcast.
func (m Model) BroadcastDue(now time.Time) bool {
	return m.channelKnown && !m.nextBroadcastAt.IsZero() && !m.nextBroadcastAt.After(now)
}

func (m Model) channelUpdated(worldId world.Id, channelId channel.Id) Model {
	m.worldId = worldId
	m.channelId = channelId
	m.channelKnown = true
	return m
}

func (m Model) skillLevelUpdated(level byte) Model {
	m.skillLevel = level
	return m
}

func (m Model) dirtyMarked(at time.Time) Model {
	m.dirtyAt = at
	return m
}

func (m Model) dirtyCleared() Model {
	m.dirtyAt = time.Time{}
	return m
}

// evaluated records a re-evaluation outcome: the captured active state, the
// refreshed character level, and the next broadcast deadline. The scheduling
// policy (reset-on-transition vs preserve-running-cadence) and the inline
// emission of state transitions live in the ticker's reevaluate (processor.go),
// which computes nextBroadcastAt; this method is a plain immutable setter.
func (m Model) evaluated(active bool, characterLevel byte, nextBroadcastAt time.Time) Model {
	m.active = active
	m.characterLevel = characterLevel
	m.nextBroadcastAt = nextBroadcastAt
	return m
}

func (m Model) broadcastScheduled(next time.Time) Model {
	m.nextBroadcastAt = next
	return m
}

type modelJSON struct {
	WorldId         world.Id   `json:"worldId"`
	ChannelId       channel.Id `json:"channelId"`
	ChannelKnown    bool       `json:"channelKnown"`
	CharacterId     uint32     `json:"characterId"`
	CharacterLevel  byte       `json:"characterLevel"`
	SkillLevel      byte       `json:"skillLevel"`
	Active          bool       `json:"active"`
	DirtyAt         time.Time  `json:"dirtyAt"`
	NextBroadcastAt time.Time  `json:"nextBroadcastAt"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		WorldId:         m.worldId,
		ChannelId:       m.channelId,
		ChannelKnown:    m.channelKnown,
		CharacterId:     m.characterId,
		CharacterLevel:  m.characterLevel,
		SkillLevel:      m.skillLevel,
		Active:          m.active,
		DirtyAt:         m.dirtyAt,
		NextBroadcastAt: m.nextBroadcastAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.worldId = aux.WorldId
	m.channelId = aux.ChannelId
	m.channelKnown = aux.ChannelKnown
	m.characterId = aux.CharacterId
	m.characterLevel = aux.CharacterLevel
	m.skillLevel = aux.SkillLevel
	m.active = aux.Active
	m.dirtyAt = aux.DirtyAt
	m.nextBroadcastAt = aux.NextBroadcastAt
	return nil
}
