package mock

import (
	"atlas-data/document"
	"atlas-data/itemmake"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc         func(s *document.Storage[string, itemmake.RestModel], r model.Provider[[]itemmake.RestModel]) error
	RegisterItemMakeFunc func(path string) error
}

var _ itemmake.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, itemmake.RestModel], r model.Provider[[]itemmake.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterItemMake(path string) error {
	if m.RegisterItemMakeFunc != nil {
		return m.RegisterItemMakeFunc(path)
	}
	return nil
}
