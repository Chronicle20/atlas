package skill

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for skill data processing
type Processor interface {
	GetSkillLevel(characterId uint32, skillId uint32) model.Provider[byte]
	GetSkill(characterId uint32, skillId uint32) model.Provider[Model]
	GetSkillsByCharacter(characterId uint32) model.Provider[[]Model]
	GetSkillsMap(characterId uint32) model.Provider[map[uint32]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new skill processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetSkillLevel returns the level of a skill for a character
// Returns 0 if the skill is not found (character doesn't have the skill)
func (p *ProcessorImpl) GetSkillLevel(characterId uint32, skillId uint32) model.Provider[byte] {
	return func() (byte, error) {
		skillProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, skillId), Extract)
		skill, err := skillProvider()
		if err != nil {
			// Skill not found is not an error - return 0 level
			p.l.Debugf("Skill %d not found for character %d, returning level 0", skillId, characterId)
			return 0, nil
		}
		return skill.Level(), nil
	}
}

// GetSkill returns the complete skill model for a character
func (p *ProcessorImpl) GetSkill(characterId uint32, skillId uint32) model.Provider[Model] {
	return func() (Model, error) {
		skillProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, skillId), Extract)
		skill, err := skillProvider()
		if err != nil {
			p.l.WithError(err).Debugf("Failed to get skill data for character %d, skill %d", characterId, skillId)
			return NewModel(skillId, 0, 0), nil
		}
		return skill, nil
	}
}

// GetSkillsByCharacter returns all skills for a character
func (p *ProcessorImpl) GetSkillsByCharacter(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		skillsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacter(characterId), Extract, model.Filters[Model]())
		skills, err := skillsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get skills for character %d", characterId)
			return nil, err
		}
		return skills, nil
	}
}

// GetSkillsMap returns all skills for a character as a map keyed by skill ID
func (p *ProcessorImpl) GetSkillsMap(characterId uint32) model.Provider[map[uint32]Model] {
	return func() (map[uint32]Model, error) {
		skills, err := p.GetSkillsByCharacter(characterId)()
		if err != nil {
			return nil, err
		}

		skillMap := make(map[uint32]Model, len(skills))
		for _, skill := range skills {
			skillMap[skill.Id()] = skill
		}
		return skillMap, nil
	}
}
