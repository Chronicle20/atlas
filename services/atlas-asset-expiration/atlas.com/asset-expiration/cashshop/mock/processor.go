package mock

import (
	"atlas-asset-expiration/cashshop"
)

type ProcessorMock struct {
	GetCompartmentsFunc func(accountId uint32) ([]cashshop.CompartmentRestModel, error)
	GetAllItemsFunc     func(accountId uint32) ([]cashshop.ItemRestModel, error)
}

var _ cashshop.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetCompartments(accountId uint32) ([]cashshop.CompartmentRestModel, error) {
	if m.GetCompartmentsFunc != nil {
		return m.GetCompartmentsFunc(accountId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetAllItems(accountId uint32) ([]cashshop.ItemRestModel, error) {
	if m.GetAllItemsFunc != nil {
		return m.GetAllItemsFunc(accountId)
	}
	return nil, nil
}
