// Package allocation implements the Player NPC script-id pool: which script
// id "branch" a deployment belongs to (PRD FR-3.2/FR-3.3), and the pure
// allocator that picks a free id out of a validated usable set (design D-1).
//
// This file is dependency-free beyond routing and libs/atlas-constants: no
// database, no HTTP client, no context. The impure usable-set builder and
// its per-tenant cache live in pool.go.
package allocation

import (
	"atlas-player-npcs/routing"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// PoolMin and PoolMax bound the entire Player NPC script-id pool (PRD
// FR-3.1). PoolMax is inclusive.
const (
	PoolMin = uint32(9901000)
	PoolMax = uint32(9906599)

	// BranchBase and BranchSize describe how a branch number maps onto the
	// id space: branch b occupies [BranchBase+b*BranchSize,
	// BranchBase+b*BranchSize+BranchSize-1]. Branch 10 is therefore
	// [9901000, 9901099] — PoolMin is the first id of branch 10, and
	// PoolMax is the last id of the final branch.
	BranchBase = uint32(9900000)
	BranchSize = uint32(100)
)

// ErrPoolExhausted is returned by Allocate when no id in usable is free.
var ErrPoolExhausted = errors.New("pool_exhausted")

// BranchFor implements the PRD FR-3.2 table for Hall of Fame maps (via
// routing.IsHallOfFameMap) and the FR-3.3 GM-deploy formula
// 26 + 4*(mapId/100000000) otherwise.
//
// For a Hall of Fame map, the branch is keyed by jobId, not by the target
// map: several distinct jobs (the five Cygnus sub-branches, in particular)
// route to the same Hall of Fame map via routing.HallOfFameMapFor, but each
// gets its own script-id branch here.
//
// Pirate is the one DIVERGENT branch (task-187 audit): wire id 500 is
// Pirate at GMS v61+ but Gm at GMS v48 (job.GetType/JobCategory alone
// cannot tell them apart -- both are raw wire ids), same as
// routing.HallOfFameMapFor. jobId is resolved through set.Job.Resolve to
// this tenant version's Identity first, and only an Identity confirmed as
// Pirate-or-descendant (job.IsAIdentity(jid, job.Pirate) -- also catches
// Brawler/Gunslinger sub-jobs, 510/520) gets branch 14. Every other job
// row stays a raw, version-stable category lookup per the
// skill-job-id-guard's own excluded-id list.
func BranchFor(set constants.SkillJobSet, jobId job.Id, mapId _map.Id) uint32 {
	if !routing.IsHallOfFameMap(mapId) {
		return 26 + 4*(uint32(mapId)/100000000)
	}

	if jid, ok := set.Job.Resolve(jobId); ok && job.IsAIdentity(jid, job.Pirate) {
		return 14
	}

	switch routing.JobCategory(jobId) {
	case uint16(job.WarriorId):
		return 10
	case uint16(job.MagicianId):
		return 11
	case uint16(job.BowmanId):
		return 12
	case uint16(job.RogueId):
		return 13
	case uint16(job.DawnWarriorStage1Id):
		return 15
	case uint16(job.BlazeWizardStage1Id):
		return 16
	case uint16(job.WindArcherStage1Id):
		return 17
	case uint16(job.NightWalkerStage1Id):
		return 18
	case uint16(job.ThunderBreakerStage1Id):
		return 19
	case uint16(job.AranStage1Id):
		return 20
	}

	// Evan (job.EvanId, 2001) shares its hundreds-category (2000) with
	// Legend, so it must be checked explicitly, ahead of falling through to
	// the Legend/Beginner/Noblesse/unclassified catch-all bucket.
	switch jobId {
	case job.EvanId:
		return 21
	case job.BeginnerId:
		return 22
	case job.NoblesseId:
		return 23
	default:
		return 24
	}
}

// BranchRange returns the inclusive [min, max] id range that belongs to
// branch.
func BranchRange(branch uint32) (uint32, uint32) {
	min := BranchBase + branch*BranchSize
	max := min + BranchSize - 1
	return min, max
}

// Allocate picks a free script id out of usable, preferring branch first
// (ascending) and falling back to the whole usable set (ascending) when the
// branch is empty or exhausted (design D-1). The fallback scans the same
// validated usable set as the branch scan, so a fallback-allocated id is
// exactly as safe as a branch-allocated one — there is no second validation
// pass and no distinct "fell back" error.
func Allocate(usable map[uint32]bool, inUse map[uint32]bool, branch uint32) (uint32, error) {
	min, max := BranchRange(branch)
	for id := min; id <= max; id++ {
		if usable[id] && !inUse[id] {
			return id, nil
		}
	}

	for id := PoolMin; id <= PoolMax; id++ {
		if usable[id] && !inUse[id] {
			return id, nil
		}
	}

	return 0, ErrPoolExhausted
}
