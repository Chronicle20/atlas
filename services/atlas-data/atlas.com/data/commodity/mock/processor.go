package mock

import (
	"atlas-data/commodity"
	"atlas-data/document"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc          func(s *document.Storage[string, commodity.RestModel], r model.Provider[[]commodity.RestModel]) error
	RegisterCommodityFunc func(path string) error
}

var _ commodity.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, commodity.RestModel], r model.Provider[[]commodity.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterCommodity(path string) error {
	if m.RegisterCommodityFunc != nil {
		return m.RegisterCommodityFunc(path)
	}
	return nil
}
