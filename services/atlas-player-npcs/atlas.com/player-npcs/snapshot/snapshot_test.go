package snapshot

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/ranking"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// stubCharacterProcessor drives Capture off a fixed character.Model
// rather than an HTTP call.
type stubCharacterProcessor struct {
	model character.Model
	err   error
}

func (s stubCharacterProcessor) GetById(characterId uint32) (character.Model, error) {
	return s.model, s.err
}

func (s stubCharacterProcessor) ByNameProvider(name string) model.Provider[[]character.Model] {
	return func() ([]character.Model, error) { return []character.Model{s.model}, s.err }
}

func (s stubCharacterProcessor) GetByName(name string) (character.Model, error) {
	return s.model, s.err
}

var _ character.Processor = stubCharacterProcessor{}

// stubRankingProcessor drives Capture off a fixed ranking.Model rather
// than an HTTP call.
type stubRankingProcessor struct {
	model ranking.Model
	err   error
}

func (s stubRankingProcessor) GetByCharacterId(characterId uint32, worldId world.Id) (ranking.Model, error) {
	return s.model, s.err
}

var _ ranking.Processor = stubRankingProcessor{}

func extractCharacter(t *testing.T, rm character.RestModel) character.Model {
	t.Helper()
	m, err := character.Extract(rm)
	if err != nil {
		t.Fatalf("character.Extract returned error: %v", err)
	}
	return m
}

func extractRanking(t *testing.T, rm ranking.RestModel) ranking.Model {
	t.Helper()
	m, err := ranking.Extract(rm)
	if err != nil {
		t.Fatalf("ranking.Extract returned error: %v", err)
	}
	return m
}

func TestCaptureSnapshot(t *testing.T) {
	t.Run("appearance", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{Gender: 0, SkinColor: 3, Face: 20000, Hair: 30030})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if snap.Gender() != 0 {
			t.Errorf("Gender() = %d, want 0", snap.Gender())
		}
		if snap.SkinColor() != 3 {
			t.Errorf("SkinColor() = %d, want 3", snap.SkinColor())
		}
		if snap.Face() != 20000 {
			t.Errorf("Face() = %d, want 20000", snap.Face())
		}
		if snap.Hair() != 30030 {
			t.Errorf("Hair() = %d, want 30030", snap.Hair())
		}
	})

	t.Run("job category", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{JobId: 112})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if uint16(snap.JobId()) != 100 {
			t.Errorf("JobId() = %d, want 100", snap.JobId())
		}
	})

	t.Run("visible equip", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{
			Equipment: []character.EquippedItemRestModel{{Slot: -5, TemplateId: 1050000}},
		})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if len(snap.Equipment()) != 1 {
			t.Fatalf("len(Equipment()) = %d, want 1", len(snap.Equipment()))
		}
		if snap.Equipment()[0].Slot() != 5 || snap.Equipment()[0].ItemId() != 1050000 {
			t.Errorf("Equipment()[0] = %+v, want slot 5 itemId 1050000", snap.Equipment()[0])
		}
	})

	t.Run("cash equip masks", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{
			Equipment: []character.EquippedItemRestModel{
				{Slot: -5, TemplateId: 1050000},   // real
				{Slot: -105, TemplateId: 1053000}, // cash, masks -5
			},
		})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if len(snap.Equipment()) != 2 {
			t.Fatalf("len(Equipment()) = %d, want 2", len(snap.Equipment()))
		}
		if snap.Equipment()[0].Slot() != 5 || snap.Equipment()[0].ItemId() != 1053000 {
			t.Errorf("visible row = %+v, want slot 5 itemId 1053000 (cash)", snap.Equipment()[0])
		}
		if snap.Equipment()[1].Slot() != 105 || snap.Equipment()[1].ItemId() != 1050000 {
			t.Errorf("masked row = %+v, want slot 105 itemId 1050000 (real)", snap.Equipment()[1])
		}
	})

	t.Run("out-of-range slot dropped", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{
			Equipment: []character.EquippedItemRestModel{{Slot: -60, TemplateId: 9999999}},
		})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if len(snap.Equipment()) != 0 {
			t.Errorf("len(Equipment()) = %d, want 0", len(snap.Equipment()))
		}
	})

	t.Run("ranks", func(t *testing.T) {
		c := extractCharacter(t, character.RestModel{})
		r := extractRanking(t, ranking.RestModel{Rank: 42, JobRank: 7})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{model: r})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if snap.WorldRank() != 42 {
			t.Errorf("WorldRank() = %d, want 42", snap.WorldRank())
		}
		if snap.OverallRank() != 42 {
			t.Errorf("OverallRank() = %d, want 42", snap.OverallRank())
		}
	})

	t.Run("no ranking", func(t *testing.T) {
		// ranking.Processor already turns a 404 into the zero-value Model
		// with a nil error (design §6.3); the stub mirrors that contract.
		c := extractCharacter(t, character.RestModel{})
		snap, err := Capture(1001, world.Id(0), stubCharacterProcessor{model: c}, stubRankingProcessor{})
		if err != nil {
			t.Fatalf("Capture returned error: %v", err)
		}
		if snap.WorldRank() != 0 {
			t.Errorf("WorldRank() = %d, want 0", snap.WorldRank())
		}
		if snap.OverallRank() != 0 {
			t.Errorf("OverallRank() = %d, want 0", snap.OverallRank())
		}
	})
}
