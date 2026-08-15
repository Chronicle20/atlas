package mock

import (
	"atlas-inventory/data/tradeability"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the tradeability reader. Following
// the Atlas mock convention (cf. atlas-trades data/item/mock), each Func field
// defaults to a zero-valued SUCCESS when left unset.
//
// Be deliberate about that here. A zero-valued tradeability.Model means
// "tradeable, no karma type" with a nil error — which is exactly the permissive
// default the real Processor refuses to produce. A consumer test that leaves
// GetFunc unset does not exercise a refusal; it exercises the tradeable path,
// silently. Set GetFunc explicitly in every karma test, including the ones whose
// subject is a gate that fires before the lookup is reached.
type ProcessorMock struct {
	ByIdProviderFunc func(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model]
	GetFunc          func(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error)
}

var _ tradeability.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(inventoryType, templateId)
	}
	return model.FixedProvider(tradeability.Model{})
}

func (m *ProcessorMock) Get(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error) {
	if m.GetFunc != nil {
		return m.GetFunc(inventoryType, templateId)
	}
	return tradeability.Model{}, nil
}
