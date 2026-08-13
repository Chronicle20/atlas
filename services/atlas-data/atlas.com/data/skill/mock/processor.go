package mock

import (
	"atlas-data/document"
	"atlas-data/skill"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc      func(s *document.Storage[string, skill.RestModel], r model.Provider[skill.Derivation]) (skill.Stats, error)
	RegisterSkillFunc func(path string) (skill.Stats, error)
}

var _ skill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, skill.RestModel], r model.Provider[skill.Derivation]) (skill.Stats, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return skill.Stats{}, nil
}

func (m *ProcessorMock) RegisterSkill(path string) (skill.Stats, error) {
	if m.RegisterSkillFunc != nil {
		return m.RegisterSkillFunc(path)
	}
	return skill.Stats{}, nil
}
