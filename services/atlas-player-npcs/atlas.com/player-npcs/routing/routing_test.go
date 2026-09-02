package routing

import (
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestHallOfFameMapFor(t *testing.T) {
	// v611 (GMS 61.1) is a provisioned post-Pirate version: wire 500 resolves
	// to job.Pirate there. v481 (GMS 48.1) is a provisioned pre-Pirate
	// version: wire 500 resolves to job.Gm there instead (task-187 audit).
	v611 := constants.For("GMS", 61, 1)
	v481 := constants.For("GMS", 48, 1)

	tests := []struct {
		name  string
		set   constants.SkillJobSet
		jobId job.Id
		want  _map.Id
	}{
		{"warrior", v611, job.WarriorId, _map.VictoriaRoadHallOfWarriors1Id},
		{"fighter (sub-job)", v611, job.Id(110), _map.VictoriaRoadHallOfWarriors1Id},
		{"magician", v611, job.MagicianId, _map.VictoriaRoadHallOfMagicians1Id},
		{"bowman", v611, job.BowmanId, _map.VictoriaRoadHallOfBowmen1Id},
		{"thief", v611, job.RogueId, _map.VictoriaRoadHallOfThieves1Id},
		{"pirate wire 500 at v61+", v611, job.PirateId, _map.TheNautilusTrainingRoom2Id},
		{"pirate wire 500 at v48 (Gm there, not Pirate)", v481, job.PirateId, _map.EmpressRoadKnightsChamber2ndFloorId},
		{"pirate sub-job (brawler) at v61+", v611, job.Id(510), _map.TheNautilusTrainingRoom2Id},
		{"dawn warrior", v611, job.Id(1100), _map.EmpressRoadKnightsChamber1Id},
		{"thunder breaker", v611, job.Id(1500), _map.EmpressRoadKnightsChamber1Id},
		{"aran", v611, job.Id(2100), _map.SnowIslandPalaceOfTheMaster1Id},
		{"beginner", v611, job.BeginnerId, _map.EmpressRoadKnightsChamber2ndFloorId},
		{"noblesse", v611, job.NoblesseId, _map.EmpressRoadKnightsChamber2ndFloorId},
		{"evan", v611, job.EvanId, _map.EmpressRoadKnightsChamber2ndFloorId},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HallOfFameMapFor(tt.set, tt.jobId)
			if got != tt.want {
				t.Errorf("HallOfFameMapFor(%v) = %v, want %v", tt.jobId, got, tt.want)
			}
		})
	}
}

func TestIsPodiumMap(t *testing.T) {
	tests := []struct {
		name  string
		mapId _map.Id
		want  bool
	}{
		{"hall of warriors", _map.VictoriaRoadHallOfWarriors1Id, true},
		{"hall of magicians", _map.VictoriaRoadHallOfMagicians1Id, true},
		{"hall of bowmen", _map.VictoriaRoadHallOfBowmen1Id, true},
		{"hall of thieves", _map.VictoriaRoadHallOfThieves1Id, true},
		{"nautilus training room", _map.TheNautilusTrainingRoom2Id, true},
		{"knights' chamber", _map.EmpressRoadKnightsChamber1Id, false},
		{"knights' chamber 2nd floor", _map.EmpressRoadKnightsChamber2ndFloorId, false},
		{"arbitrary map", _map.Id(100000000), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPodiumMap(tt.mapId)
			if got != tt.want {
				t.Errorf("IsPodiumMap(%v) = %v, want %v", tt.mapId, got, tt.want)
			}
		})
	}
}

func TestIsHallOfFameMap(t *testing.T) {
	hallOfFameMaps := []_map.Id{
		_map.VictoriaRoadHallOfWarriors1Id,
		_map.VictoriaRoadHallOfMagicians1Id,
		_map.VictoriaRoadHallOfBowmen1Id,
		_map.VictoriaRoadHallOfThieves1Id,
		_map.TheNautilusTrainingRoom2Id,
		_map.EmpressRoadKnightsChamber1Id,
		_map.EmpressRoadKnightsChamber2Id,
		_map.EmpressRoadKnightsChamber2ndFloorId,
		_map.EmpressRoadKnightsChamber3rdFloorId,
		_map.SnowIslandPalaceOfTheMaster1Id,
	}

	for _, mapId := range hallOfFameMaps {
		t.Run(fmt.Sprintf("%d", mapId), func(t *testing.T) {
			if !IsHallOfFameMap(mapId) {
				t.Errorf("IsHallOfFameMap(%v) = false, want true", mapId)
			}
		})
	}

	t.Run("arbitrary map", func(t *testing.T) {
		if IsHallOfFameMap(_map.Id(100000000)) {
			t.Errorf("IsHallOfFameMap(100000000) = true, want false")
		}
	})
}
