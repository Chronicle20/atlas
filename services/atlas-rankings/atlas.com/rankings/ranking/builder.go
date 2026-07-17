package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	characterId     uint32
	worldId         world.Id
	jobCategory     uint16
	overallRank     uint32
	overallRankMove int32
	jobRank         uint32
	jobRankMove     int32
	computedAt      time.Time
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetCharacterId(v uint32) *Builder    { b.characterId = v; return b }
func (b *Builder) SetWorldId(v world.Id) *Builder      { b.worldId = v; return b }
func (b *Builder) SetJobCategory(v uint16) *Builder    { b.jobCategory = v; return b }
func (b *Builder) SetOverallRank(v uint32) *Builder    { b.overallRank = v; return b }
func (b *Builder) SetOverallRankMove(v int32) *Builder { b.overallRankMove = v; return b }
func (b *Builder) SetJobRank(v uint32) *Builder        { b.jobRank = v; return b }
func (b *Builder) SetJobRankMove(v int32) *Builder     { b.jobRankMove = v; return b }
func (b *Builder) SetComputedAt(v time.Time) *Builder  { b.computedAt = v; return b }

func (b *Builder) Build() Model {
	return Model{
		characterId:     b.characterId,
		worldId:         b.worldId,
		jobCategory:     b.jobCategory,
		overallRank:     b.overallRank,
		overallRankMove: b.overallRankMove,
		jobRank:         b.jobRank,
		jobRankMove:     b.jobRankMove,
		computedAt:      b.computedAt,
	}
}
