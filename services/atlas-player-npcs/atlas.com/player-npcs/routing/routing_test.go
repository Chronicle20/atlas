package routing

import (
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestHallOfFameMapFor(t *testing.T) {
	tests := []struct {
		name  string
		jobId job.Id
		want  _map.Id
	}{
		{"warrior", job.WarriorId, _map.VictoriaRoadHallOfWarriors1Id},
		{"fighter (sub-job)", job.Id(110), _map.VictoriaRoadHallOfWarriors1Id},
		{"magician", job.MagicianId, _map.VictoriaRoadHallOfMagicians1Id},
		{"bowman", job.BowmanId, _map.VictoriaRoadHallOfBowmen1Id},
		{"thief", job.RogueId, _map.VictoriaRoadHallOfThieves1Id},
		{"pirate", job.PirateId, _map.TheNautilusTrainingRoom2Id},
		{"dawn warrior", job.Id(1100), _map.EmpressRoadKnightsChamber1Id},
		{"thunder breaker", job.Id(1500), _map.EmpressRoadKnightsChamber1Id},
		{"aran", job.Id(2100), _map.SnowIslandPalaceOfTheMaster1Id},
		{"beginner", job.BeginnerId, _map.EmpressRoadKnightsChamber2ndFloorId},
		{"noblesse", job.NoblesseId, _map.EmpressRoadKnightsChamber2ndFloorId},
		{"evan", job.EvanId, _map.EmpressRoadKnightsChamber2ndFloorId},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HallOfFameMapFor(tt.jobId)
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
