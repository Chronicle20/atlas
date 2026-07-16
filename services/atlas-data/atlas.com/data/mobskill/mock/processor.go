package mock

import (
	"atlas-data/document"
	"atlas-data/mobskill"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc         func(s *document.Storage[string, mobskill.RestModel], r model.Provider[[]mobskill.RestModel]) error
	RegisterMobSkillFunc func(path string) error
}

var _ mobskill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, mobskill.RestModel], r model.Provider[[]mobskill.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterMobSkill(path string) error {
	if m.RegisterMobSkillFunc != nil {
		return m.RegisterMobSkillFunc(path)
	}
	return nil
}
