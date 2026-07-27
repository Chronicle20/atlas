package mock

import (
	"atlas-data/document"
	"atlas-data/equipment"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc          func(tx *gorm.DB, s *document.Storage[string, equipment.RestModel], r model.Provider[equipment.RestModel]) error
	RegisterEquipmentFunc func(path string) error
}

var _ equipment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(tx *gorm.DB, s *document.Storage[string, equipment.RestModel], r model.Provider[equipment.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(tx, s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterEquipment(path string) error {
	if m.RegisterEquipmentFunc != nil {
		return m.RegisterEquipmentFunc(path)
	}
	return nil
}
