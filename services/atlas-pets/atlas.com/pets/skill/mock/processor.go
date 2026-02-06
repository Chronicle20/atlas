package mock

import (
	"atlas-pets/skill"

	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-model/model"
)

type Processor struct {
	ByCharacterIdProviderFn func(characterId uint32) model.Provider[[]skill.Model]
	GetByCharacterIdFn      func(characterId uint32) ([]skill.Model, error)
	HasSkillFn              func(characterId uint32, ids ...skill2.Id) bool
}

func (m *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]skill.Model] {
	if m.ByCharacterIdProviderFn != nil {
		return m.ByCharacterIdProviderFn(characterId)
	}
	return nil
}

func (m *Processor) GetByCharacterId(characterId uint32) ([]skill.Model, error) {
	if m.GetByCharacterIdFn != nil {
		return m.GetByCharacterIdFn(characterId)
	}
	return nil, nil
}

func (m *Processor) HasSkill(characterId uint32, ids ...skill2.Id) bool {
	if m.HasSkillFn != nil {
		return m.HasSkillFn(characterId, ids...)
	}
	return false
}
