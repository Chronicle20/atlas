package item

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the atlas-data item-information REST client.
type Processor interface {
	// TradeBlockProvider resolves the WZ tradeBlock flag of one item template,
	// reading the atlas-data resource that owns the given compartment.
	TradeBlockProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[bool]
	// TradeBlock reports whether the item template forbids trading. An error
	// means the lookup FAILED; callers must treat that as a refusal, never as
	// "tradeable" (design §7).
	TradeBlock(inventoryType inventory.Type, templateId item.Id) (bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) TradeBlockProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[bool] {
	switch inventoryType {
	case inventory.TypeValueEquip:
		return requests.Provider[EquipmentRestModel, bool](p.l, p.ctx)(requestEquipment(templateId), extractTradeBlock)
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, bool](p.l, p.ctx)(requestConsumable(templateId), extractTradeBlock)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, bool](p.l, p.ctx)(requestSetup(templateId), extractTradeBlock)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, bool](p.l, p.ctx)(requestEtc(templateId), extractTradeBlock)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, bool](p.l, p.ctx)(requestCash(templateId), extractTradeBlock)
	default:
		return model.ErrorProvider[bool](fmt.Errorf("item: no atlas-data resource for inventory type [%d]", inventoryType))
	}
}

func (p *ProcessorImpl) TradeBlock(inventoryType inventory.Type, templateId item.Id) (bool, error) {
	return p.TradeBlockProvider(inventoryType, templateId)()
}
