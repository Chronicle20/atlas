package mock

import (
	"atlas-query-aggregator/skill"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the skill.Processor interface
type ProcessorImpl struct {
	GetSkillLevelFunc        func(characterId uint32, skillId uint32) model.Provider[byte]
	GetSkillFunc             func(characterId uint32, skillId uint32) model.Provider[skill.Model]
	GetSkillsByCharacterFunc func(characterId uint32) model.Provider[[]skill.Model]
	GetSkillsMapFunc         func(characterId uint32) model.Provider[map[uint32]skill.Model]
}

// GetSkillLevel returns the level of a skill for a character
func (m *ProcessorImpl) GetSkillLevel(characterId uint32, skillId uint32) model.Provider[byte] {
	if m.GetSkillLevelFunc != nil {
		return m.GetSkillLevelFunc(characterId, skillId)
	}
	return func() (byte, error) {
		return 0, nil
	}
}

// GetSkill returns the complete skill model for a character
func (m *ProcessorImpl) GetSkill(characterId uint32, skillId uint32) model.Provider[skill.Model] {
	if m.GetSkillFunc != nil {
		return m.GetSkillFunc(characterId, skillId)
	}
	return func() (skill.Model, error) {
		return skill.NewModel(skillId, 0, 0), nil
	}
}

// GetSkillsByCharacter returns all skills for a character
func (m *ProcessorImpl) GetSkillsByCharacter(characterId uint32) model.Provider[[]skill.Model] {
	if m.GetSkillsByCharacterFunc != nil {
		return m.GetSkillsByCharacterFunc(characterId)
	}
	return func() ([]skill.Model, error) {
		return nil, nil
	}
}

// GetSkillsMap returns all skills for a character as a map keyed by skill ID
func (m *ProcessorImpl) GetSkillsMap(characterId uint32) model.Provider[map[uint32]skill.Model] {
	if m.GetSkillsMapFunc != nil {
		return m.GetSkillsMapFunc(characterId)
	}
	return func() (map[uint32]skill.Model, error) {
		return make(map[uint32]skill.Model), nil
	}
}
