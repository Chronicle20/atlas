package tradeability

import (
	"context"
	"errors"
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
		return requests.Provider[EquipmentRestModel, Model](p.l, p.ctx)(requestEquipment(p.ctx, templateId), extract)
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, Model](p.l, p.ctx)(requestConsumable(p.ctx, templateId), extract)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, Model](p.l, p.ctx)(requestSetup(p.ctx, templateId), extract)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, Model](p.l, p.ctx)(requestEtc(p.ctx, templateId), extract)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, Model](p.l, p.ctx)(requestCash(p.ctx, templateId), extract)
	default:
		return model.ErrorProvider[Model](fmt.Errorf("tradeability: no atlas-data resource for inventory type [%d]", inventoryType))
	}
}

func (p *ProcessorImpl) Get(inventoryType inventory.Type, templateId item.Id) (Model, error) {
	m, err := p.ByIdProvider(inventoryType, templateId)()
	if err != nil {
		// Diagnosis only: both arms still refuse (return the error). A 404
		// means atlas-data has no entry for this template id in this
		// compartment's resource; anything else (transport failure, decode
		// failure, 5xx) means atlas-data itself could not be reached or
		// answered. Karma gates must refuse on either — this distinction
		// exists only so an operator reading logs can tell "unreachable"
		// from "no data" without re-deriving it from the raw error text.
		if errors.Is(err, requests.ErrNotFound) {
			p.l.WithError(err).Warnf("tradeability: no atlas-data entry for inventory type [%d] template [%d] (404).", inventoryType, templateId)
		} else {
			p.l.WithError(err).Errorf("tradeability: atlas-data lookup failed for inventory type [%d] template [%d] (non-404).", inventoryType, templateId)
		}
		return Model{}, err
	}
	return m, nil
}
