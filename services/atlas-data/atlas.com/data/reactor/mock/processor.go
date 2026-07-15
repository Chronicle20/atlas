package mock

import (
	"atlas-data/reactor"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc        func(s *reactor.Storage, r model.Provider[reactor.RestModel]) error
	RegisterReactorFunc func(path string) error
}

var _ reactor.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *reactor.Storage, r model.Provider[reactor.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterReactor(path string) error {
	if m.RegisterReactorFunc != nil {
		return m.RegisterReactorFunc(path)
	}
	return nil
}
