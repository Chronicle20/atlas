package mock

import (
	"atlas-data/document"
	"atlas-data/setup"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc      func(s *document.Storage[string, setup.RestModel], r model.Provider[[]setup.RestModel]) error
	RegisterSetupFunc func(path string) error
}

var _ setup.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, setup.RestModel], r model.Provider[[]setup.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterSetup(path string) error {
	if m.RegisterSetupFunc != nil {
		return m.RegisterSetupFunc(path)
	}
	return nil
}
