package mock

import (
	"atlas-data/npc"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc    func(s *npc.Storage, r model.Provider[npc.RestModel]) error
	RegisterNpcFunc func(path string) error
}

var _ npc.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *npc.Storage, r model.Provider[npc.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterNpc(path string) error {
	if m.RegisterNpcFunc != nil {
		return m.RegisterNpcFunc(path)
	}
	return nil
}
