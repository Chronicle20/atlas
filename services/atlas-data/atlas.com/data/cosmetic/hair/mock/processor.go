package mock

import (
	"atlas-data/cosmetic/hair"
	"atlas-data/document"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc     func(s *document.Storage[string, hair.RestModel], r model.Provider[hair.RestModel]) error
	RegisterHairFunc func(path string) error
}

var _ hair.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, hair.RestModel], r model.Provider[hair.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterHair(path string) error {
	if m.RegisterHairFunc != nil {
		return m.RegisterHairFunc(path)
	}
	return nil
}
