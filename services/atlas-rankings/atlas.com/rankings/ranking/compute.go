package ranking

import (
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Input is one eligible (non-GM) character snapshot. Eligibility filtering
// (gm > 0 excluded entirely) happens before Inputs are built.
type Input struct {
	CharacterId uint32
	WorldId     world.Id
	JobId       job.Id
	Level       byte
	Experience  uint32
}

// Ranked is the computed placement for one character. Ranks are 1-based and
// unique within their scope: the characterId tiebreak makes the order a
// strict total order, so dense and ordinal ranking coincide.
type Ranked struct {
	CharacterId uint32
	WorldId     world.Id
	JobCategory uint16
	OverallRank uint32
	JobRank     uint32
}

// JobCategory buckets a job id into its top-level job division: jobId / 100.
// 0=beginner, 1=warrior, 2=magician, 3=bowman, 4=thief, 5=pirate;
// Cygnus (10-15) and Aran (20-21) fall out of the same division.
func JobCategory(jobId job.Id) uint16 {
	return uint16(jobId / 100)
}

func less(a Input, b Input) bool {
	if a.Level != b.Level {
		return a.Level > b.Level
	}
	if a.Experience != b.Experience {
		return a.Experience > b.Experience
	}
	return a.CharacterId < b.CharacterId
}

// Rank computes per-world overall and job-category placements ordered by
// level DESC, experience DESC, characterId ASC. Job ranks reuse the same
// sorted order restricted to each category.
func Rank(inputs []Input) []Ranked {
	byWorld := make(map[world.Id][]Input)
	for _, i := range inputs {
		byWorld[i.WorldId] = append(byWorld[i.WorldId], i)
	}

	results := make([]Ranked, 0, len(inputs))
	for wid, ws := range byWorld {
		sort.Slice(ws, func(i, j int) bool { return less(ws[i], ws[j]) })

		jobPos := make(map[uint16]uint32)
		for idx, c := range ws {
			cat := JobCategory(c.JobId)
			jobPos[cat]++
			results = append(results, Ranked{
				CharacterId: c.CharacterId,
				WorldId:     wid,
				JobCategory: cat,
				OverallRank: uint32(idx + 1),
				JobRank:     jobPos[cat],
			})
		}
	}
	return results
}

// Move is previousRank − newRank (positive = moved up). A character with no
// previous entry (prev == 0; 0 is never a stored rank) moves 0.
func Move(prev uint32, next uint32) int32 {
	if prev == 0 {
		return 0
	}
	return int32(prev) - int32(next)
}
