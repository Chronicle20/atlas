package mock

import (
	"atlas-data/document"
	"atlas-data/etc"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc    func(s *document.Storage[string, etc.RestModel], r model.Provider[[]etc.RestModel]) error
	RegisterEtcFunc func(path string) error
}

var _ etc.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, etc.RestModel], r model.Provider[[]etc.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterEtc(path string) error {
	if m.RegisterEtcFunc != nil {
		return m.RegisterEtcFunc(path)
	}
	return nil
}
