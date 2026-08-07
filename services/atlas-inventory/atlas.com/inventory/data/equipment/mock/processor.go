package mock

import (
	"atlas-inventory/data/equipment"
)

type ProcessorMock struct {
	DestinationSlotProviderFunc func(suggested int16) equipment.DestinationProvider
}

var _ equipment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) DestinationSlotProvider(suggested int16) equipment.DestinationProvider {
	if m.DestinationSlotProviderFunc != nil {
		return m.DestinationSlotProviderFunc(suggested)
	}
	return equipment.DestinationProvider(nil)
}
