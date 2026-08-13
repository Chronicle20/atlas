package tradeability

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor answers the two WZ questions the karma gates ask of a target item.
// An error means the LOOKUP FAILED; every caller must treat that as a refusal,
// never as a permissive default.
type Processor interface {
	ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[Model]
	Get(inventoryType inventory.Type, templateId item.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[Model] {
	switch inventoryType {
	case inventory.TypeValueEquip:
		return requests.Provider[EquipmentRestModel, Model](p.l, p.ctx)(requestEquipment(templateId), extract)
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, Model](p.l, p.ctx)(requestConsumable(templateId), extract)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, Model](p.l, p.ctx)(requestSetup(templateId), extract)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, Model](p.l, p.ctx)(requestEtc(templateId), extract)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, Model](p.l, p.ctx)(requestCash(templateId), extract)
	default:
		return model.ErrorProvider[Model](fmt.Errorf("tradeability: no atlas-data resource for inventory type [%d]", inventoryType))
	}
}

func (p *ProcessorImpl) Get(inventoryType inventory.Type, templateId item.Id) (Model, error) {
	return p.ByIdProvider(inventoryType, templateId)()
}
