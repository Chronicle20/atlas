// Package routing maps a character's job to the Hall of Fame map its Player
// NPC belongs in, and answers which Hall of Fame maps use the podium
// positioner. It is a pure, dependency-free lookup over
// libs/atlas-constants values; it holds no state and calls no other
// service.
package routing

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// JobCategory groups a job id by its hundreds-branch: (jobId / 100) * 100.
// e.g. FighterId (110) and CrusaderId (111) both belong to category 100
// (WarriorId); AranStage2Id (2110) belongs to category 2100 (AranStage1Id).
func JobCategory(jobId job.Id) uint16 {
	return (uint16(jobId) / 100) * 100
}

// hallOfFameMaps is the FR-2.2 set: every map that is a valid Player NPC
// deployment target, whether or not it is an automatic-deployment
// destination (FR-2.1). All ten constants already exist in
// libs/atlas-constants/map/constants.go (design §1 C-2).
var hallOfFameMaps = map[_map.Id]struct{}{
	_map.VictoriaRoadHallOfWarriors1Id:       {},
	_map.VictoriaRoadHallOfMagicians1Id:      {},
	_map.VictoriaRoadHallOfBowmen1Id:         {},
	_map.VictoriaRoadHallOfThieves1Id:        {},
	_map.TheNautilusTrainingRoom2Id:          {},
	_map.EmpressRoadKnightsChamber1Id:        {},
	_map.EmpressRoadKnightsChamber2Id:        {},
	_map.EmpressRoadKnightsChamber2ndFloorId: {},
	_map.EmpressRoadKnightsChamber3rdFloorId: {},
	_map.SnowIslandPalaceOfTheMaster1Id:      {},
}

// podiumMaps is the FR-2.3 set: the five maps that use the podium
// positioner rather than the grid positioner. Every other Hall of Fame map
// uses the grid positioner.
var podiumMaps = map[_map.Id]struct{}{
	_map.VictoriaRoadHallOfWarriors1Id:  {},
	_map.VictoriaRoadHallOfMagicians1Id: {},
	_map.VictoriaRoadHallOfBowmen1Id:    {},
	_map.VictoriaRoadHallOfThieves1Id:   {},
	_map.TheNautilusTrainingRoom2Id:     {},
}

// HallOfFameMapFor returns the automatic-deployment target map for jobId,
// per FR-2.1:
//
//   - Explorer branches (Warrior/Magician/Bowman/Thief/Pirate) route by
//     JobCategory to their branch's Hall.
//   - The Aran branch (all AranStageNId) routes to the Palace of the
//     Master; it shares JobCategory (2100) across every stage, and is
//     distinct from Legend/Evan (JobCategory 2000/2200), which fall
//     through to the default.
//   - Any Cygnus job (job.GetType == job.TypeCygnus) routes to Knights'
//     Chamber.
//   - Everything else (Beginner, Noblesse, Legend, Evan, unclassified)
//     defaults to Knights' Chamber 2nd Floor.
func HallOfFameMapFor(jobId job.Id) _map.Id {
	switch JobCategory(jobId) {
	case uint16(job.WarriorId):
		return _map.VictoriaRoadHallOfWarriors1Id
	case uint16(job.MagicianId):
		return _map.VictoriaRoadHallOfMagicians1Id
	case uint16(job.BowmanId):
		return _map.VictoriaRoadHallOfBowmen1Id
	case uint16(job.RogueId):
		return _map.VictoriaRoadHallOfThieves1Id
	case uint16(job.PirateId):
		return _map.TheNautilusTrainingRoom2Id
	case uint16(job.AranStage1Id):
		return _map.SnowIslandPalaceOfTheMaster1Id
	}

	// NoblesseId (1000) is the Cygnus line's own "unclassified beginner" job
	// and would otherwise satisfy job.GetType == job.TypeCygnus below; treat
	// it, like every other beginner job, as unclassified instead.
	if !job.IsBeginner(jobId) && job.GetType(jobId) == job.TypeCygnus {
		return _map.EmpressRoadKnightsChamber1Id
	}

	return _map.EmpressRoadKnightsChamber2ndFloorId
}

// IsPodiumMap reports whether mapId is one of the five FR-2.3 podium maps
// (podium positioner). All other maps use the grid positioner.
func IsPodiumMap(mapId _map.Id) bool {
	_, ok := podiumMaps[mapId]
	return ok
}

// IsHallOfFameMap reports whether mapId is one of the ten FR-2.2 Hall of
// Fame maps — valid Player NPC deployment targets, whether or not they are
// an automatic-deployment destination.
func IsHallOfFameMap(mapId _map.Id) bool {
	_, ok := hallOfFameMaps[mapId]
	return ok
}
