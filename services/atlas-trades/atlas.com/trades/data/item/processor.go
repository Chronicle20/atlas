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
	// SlotMax reports how many of the template fit in one inventory slot. An
	// error means the lookup FAILED; as with TradeBlock, callers refuse rather
	// than assume a value.
	SlotMax(inventoryType inventory.Type, templateId item.Id) (uint32, error)
	// SlotMaxProvider resolves the WZ slotMax of one item template, reading the
	// atlas-data resource that owns the given compartment.
	SlotMaxProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[uint32]
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
		return requests.Provider[EquipmentRestModel, bool](p.l, p.ctx)(requestEquipment(p.ctx, templateId), extractTradeBlock)
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, bool](p.l, p.ctx)(requestConsumable(p.ctx, templateId), extractTradeBlock)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, bool](p.l, p.ctx)(requestSetup(p.ctx, templateId), extractTradeBlock)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, bool](p.l, p.ctx)(requestEtc(p.ctx, templateId), extractTradeBlock)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, bool](p.l, p.ctx)(requestCash(p.ctx, templateId), extractTradeBlock)
	default:
		return model.ErrorProvider[bool](fmt.Errorf("item: no atlas-data resource for inventory type [%d]", inventoryType))
	}
}

func (p *ProcessorImpl) TradeBlock(inventoryType inventory.Type, templateId item.Id) (bool, error) {
	return p.TradeBlockProvider(inventoryType, templateId)()
}

// SlotMax resolves the WZ slotMax of one item template.
//
// EQUIP short-circuits to 1 without a request: atlas-data's equipment resource
// carries no slotMax, and an equip never merges into an existing stack anyway
// (atlas-inventory's Accept gates its merge on the incoming asset having a
// quantity). Returning 1 makes every equip read as "a stack that is already
// full", which is exactly how the free-slot check must treat it.
//
// A slotMax of 0 from atlas-data is coerced to 1 for the same reason: a zero
// would otherwise read as "no stack ever has room", which is the same answer,
// but by way of arithmetic that also makes a merge look like an overflow.
func (p *ProcessorImpl) SlotMax(inventoryType inventory.Type, templateId item.Id) (uint32, error) {
	if inventoryType == inventory.TypeValueEquip {
		return 1, nil
	}
	max, err := p.SlotMaxProvider(inventoryType, templateId)()
	if err != nil {
		return 0, err
	}
	if max == 0 {
		return 1, nil
	}
	return max, nil
}

// SlotMaxProvider resolves the WZ slotMax of one item template from the
// atlas-data resource that owns the given compartment.
func (p *ProcessorImpl) SlotMaxProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[uint32] {
	switch inventoryType {
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, uint32](p.l, p.ctx)(requestConsumable(p.ctx, templateId), extractSlotMax)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, uint32](p.l, p.ctx)(requestSetup(p.ctx, templateId), extractSlotMax)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, uint32](p.l, p.ctx)(requestEtc(p.ctx, templateId), extractSlotMax)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, uint32](p.l, p.ctx)(requestCash(p.ctx, templateId), extractSlotMax)
	default:
		return model.ErrorProvider[uint32](fmt.Errorf("item: no atlas-data slotMax resource for inventory type [%d]", inventoryType))
	}
}
