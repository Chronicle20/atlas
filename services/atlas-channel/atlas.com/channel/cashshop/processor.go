package cashshop

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	compartment2 "atlas-channel/compartment"
	"atlas-channel/kafka/message/cashshop"
	compartment3 "atlas-channel/kafka/message/compartment"
	"atlas-channel/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) Enter(characterId uint32, m _map.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(CharacterEnterCashShopStatusEventProvider(characterId, m))
}

func (p *Processor) Exit(characterId uint32, m _map.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(CharacterExitCashShopStatusEventProvider(characterId, m))
}

type PointType string

const (
	PointTypeCredit  = "CREDIT"
	PointTypeMaple   = "POINTS"
	PointTypePrepaid = "PREPAID"
)

func GetPointType(arg bool) PointType {
	if arg {
		return PointTypeMaple
	}
	return PointTypeCredit
}

func (p *Processor) RequestInventoryIncreasePurchaseByType(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error {
	p.l.Debugf("Character [%d] purchasing inventory [%d] expansion using currency [%d].", characterId, inventoryType, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByTypeCommandProvider(characterId, currency, inventoryType))
}

func (p *Processor) RequestInventoryIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing inventory expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *Processor) RequestStorageIncreasePurchase(characterId uint32, isPoints bool, currency uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion using currency [%d].", characterId, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseCommandProvider(characterId, currency))
}

func (p *Processor) RequestStorageIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *Processor) RequestCharacterSlotIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing character slot expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCharacterSlotIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *Processor) RequestPurchase(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error {
	p.l.Debugf("Character [%d] purchasing [%d] with currency [%d], zero [%d]", characterId, serialNumber, currency, zero)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestPurchaseCommandProvider(characterId, serialNumber, currency))
}

func (p *Processor) MoveFromCashInventory(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error {
	p.l.Infof("Character [%d] moving [%d] to inventory [%d] to slot [%d].", characterId, serialNumber, inventoryType, slot)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(MoveFromCashInventoryCommandProvider(characterId, serialNumber, inventoryType, slot))
}

func (p *Processor) MoveToCashInventory(characterId uint32, serialNumber uint64, inventoryType byte) error {
	p.l.Infof("Character [%d] moving [%d] from inventory [%d] to cash inventory.", characterId, serialNumber, inventoryType)

	// Get the character's inventory
	cp := character.NewProcessor(p.l, p.ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get character [%d] inventory.", characterId)
		return err
	}

	// Get the compartment
	compartment := c.Inventory().CompartmentByType(inventory.Type(inventoryType))

	// Check if the compartment has the asset with CashId matching the serialNumber
	var foundAsset *asset.Model[any]
	for _, asset := range compartment.Assets() {
		// Check if the asset is cash-equipable, cash, or pet
		if asset.IsCashEquipable() {
			if cashEquipable, ok := asset.ReferenceData().(interface{ GetCashId() int64 }); ok {
				if uint64(cashEquipable.GetCashId()) == serialNumber {
					assetCopy := asset
					foundAsset = &assetCopy
					break
				}
			}
		} else if asset.IsCash() {
			if cash, ok := asset.ReferenceData().(interface{ CashId() int64 }); ok {
				if uint64(cash.CashId()) == serialNumber {
					assetCopy := asset
					foundAsset = &assetCopy
					break
				}
			}
		} else if asset.IsPet() {
			if pet, ok := asset.ReferenceData().(interface{ CashId() uint64 }); ok {
				if pet.CashId() == serialNumber {
					assetCopy := asset
					foundAsset = &assetCopy
					break
				}
			}
		}
	}

	if foundAsset == nil {
		p.l.Errorf("Character [%d] does not have asset with CashId [%d] in inventory [%d].", characterId, serialNumber, inventoryType)
		return errors.New("asset not found")
	}

	return compartment2.NewProcessor(p.l, p.ctx).MoveToOtherInventory(characterId, inventory.Type(inventoryType), foundAsset.Slot(), compartment3.CashInventoryType)
}
