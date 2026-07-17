package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
	characterId     uint32
	worldId         world.Id
	jobCategory     uint16
	overallRank     uint32
	overallRankMove int32
	jobRank         uint32
	jobRankMove     int32
	computedAt      time.Time
}

func (m Model) CharacterId() uint32    { return m.characterId }
func (m Model) WorldId() world.Id      { return m.worldId }
func (m Model) JobCategory() uint16    { return m.jobCategory }
func (m Model) OverallRank() uint32    { return m.overallRank }
func (m Model) OverallRankMove() int32 { return m.overallRankMove }
func (m Model) JobRank() uint32        { return m.jobRank }
func (m Model) JobRankMove() int32     { return m.jobRankMove }
func (m Model) ComputedAt() time.Time  { return m.computedAt }

type CycleModel struct {
	lastStartedAt    time.Time
	lastCompletedAt  *time.Time
	charactersRanked uint32
	durationMs       uint32
}

func (m CycleModel) LastStartedAt() time.Time    { return m.lastStartedAt }
func (m CycleModel) LastCompletedAt() *time.Time { return m.lastCompletedAt }
func (m CycleModel) CharactersRanked() uint32    { return m.charactersRanked }
func (m CycleModel) DurationMs() uint32          { return m.durationMs }
