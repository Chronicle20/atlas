package skill_test

import (
	"atlas-query-aggregator/skill"
	"atlas-query-aggregator/skill/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_GetSkillLevel_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillLevelFunc: func(characterId uint32, skillId uint32) model.Provider[byte] {
			return func() (byte, error) {
				if skillId == 1001 {
					return 20, nil
				}
				return 0, nil
			}
		},
	}

	level, err := mockProcessor.GetSkillLevel(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if level != 20 {
		t.Errorf("Expected level=20, got %d", level)
	}
}

func TestProcessorMock_GetSkillLevel_NotFound(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillLevelFunc: func(characterId uint32, skillId uint32) model.Provider[byte] {
			return func() (byte, error) {
				return 0, nil // Skill not found returns 0, not error
			}
		},
	}

	level, err := mockProcessor.GetSkillLevel(123, 9999)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if level != 0 {
		t.Errorf("Expected level=0 for not found skill, got %d", level)
	}
}

func TestProcessorMock_GetSkill_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillFunc: func(characterId uint32, skillId uint32) model.Provider[skill.Model] {
			return func() (skill.Model, error) {
				return skill.NewModel(skillId, 20, 30), nil
			}
		},
	}

	s, err := mockProcessor.GetSkill(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if s.Id() != 1001 {
		t.Errorf("Expected Id=1001, got %d", s.Id())
	}

	if s.Level() != 20 {
		t.Errorf("Expected Level=20, got %d", s.Level())
	}

	if s.MasterLevel() != 30 {
		t.Errorf("Expected MasterLevel=30, got %d", s.MasterLevel())
	}
}

func TestProcessorMock_GetSkillsByCharacter_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillsByCharacterFunc: func(characterId uint32) model.Provider[[]skill.Model] {
			return func() ([]skill.Model, error) {
				return []skill.Model{
					skill.NewModel(1001, 10, 20),
					skill.NewModel(1002, 5, 10),
					skill.NewModel(1003, 1, 5),
				}, nil
			}
		},
	}

	skills, err := mockProcessor.GetSkillsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("Expected 3 skills, got %d", len(skills))
	}
}

func TestProcessorMock_GetSkillsByCharacter_Empty(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillsByCharacterFunc: func(characterId uint32) model.Provider[[]skill.Model] {
			return func() ([]skill.Model, error) {
				return []skill.Model{}, nil
			}
		},
	}

	skills, err := mockProcessor.GetSkillsByCharacter(456)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestProcessorMock_GetSkillsByCharacter_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillsByCharacterFunc: func(characterId uint32) model.Provider[[]skill.Model] {
			return func() ([]skill.Model, error) {
				return nil, errors.New("skill service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetSkillsByCharacter(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_GetSkillsMap_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSkillsMapFunc: func(characterId uint32) model.Provider[map[uint32]skill.Model] {
			return func() (map[uint32]skill.Model, error) {
				return map[uint32]skill.Model{
					1001: skill.NewModel(1001, 10, 20),
					1002: skill.NewModel(1002, 5, 10),
				}, nil
			}
		},
	}

	skillMap, err := mockProcessor.GetSkillsMap(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(skillMap) != 2 {
		t.Errorf("Expected 2 skills in map, got %d", len(skillMap))
	}

	if s, ok := skillMap[1001]; !ok {
		t.Error("Expected skill 1001 in map")
	} else if s.Level() != 10 {
		t.Errorf("Expected skill 1001 level=10, got %d", s.Level())
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetSkillLevel returns 0
	level, err := mockProcessor.GetSkillLevel(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error from default GetSkillLevel, got %v", err)
	}

	if level != 0 {
		t.Errorf("Expected default level=0, got %d", level)
	}

	// Test default GetSkill returns model with level 0
	s, err := mockProcessor.GetSkill(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error from default GetSkill, got %v", err)
	}

	if s.Level() != 0 {
		t.Errorf("Expected default skill level=0, got %d", s.Level())
	}

	// Test default GetSkillsByCharacter returns nil
	skills, err := mockProcessor.GetSkillsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetSkillsByCharacter, got %v", err)
	}

	if skills != nil {
		t.Errorf("Expected default skills=nil, got %v", skills)
	}

	// Test default GetSkillsMap returns empty map
	skillMap, err := mockProcessor.GetSkillsMap(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetSkillsMap, got %v", err)
	}

	if len(skillMap) != 0 {
		t.Errorf("Expected default skillMap to be empty, got %d", len(skillMap))
	}
}
