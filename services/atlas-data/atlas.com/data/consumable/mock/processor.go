package mock

import (
	"atlas-data/consumable"
	"atlas-data/document"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc           func(s *document.Storage[string, consumable.RestModel], r model.Provider[[]consumable.RestModel]) error
	RegisterConsumableFunc func(path string) error
}

var _ consumable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, consumable.RestModel], r model.Provider[[]consumable.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterConsumable(path string) error {
	if m.RegisterConsumableFunc != nil {
		return m.RegisterConsumableFunc(path)
	}
	return nil
}
