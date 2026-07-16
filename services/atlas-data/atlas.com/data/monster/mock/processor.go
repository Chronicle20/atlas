package mock

import (
	"atlas-data/monster"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc        func(s *monster.Storage, r model.Provider[monster.RestModel]) error
	RegisterMonsterFunc func(path string) error
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *monster.Storage, r model.Provider[monster.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterMonster(path string) error {
	if m.RegisterMonsterFunc != nil {
		return m.RegisterMonsterFunc(path)
	}
	return nil
}
