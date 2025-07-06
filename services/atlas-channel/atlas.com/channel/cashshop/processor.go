package cashshop

import (
	"atlas-channel/asset"
	asset2 "atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/inventory/compartment"
	compartment2 "atlas-channel/compartment"
	"atlas-channel/kafka/message/cashshop"
	"atlas-channel/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for cashshop processing
type Processor interface {
	Enter(characterId uint32, m _map.Model) error
	Exit(characterId uint32, m _map.Model) error
	RequestInventoryIncreasePurchaseByType(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error
	RequestInventoryIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error
	RequestStorageIncreasePurchase(characterId uint32, isPoints bool, currency uint32) error
	RequestStorageIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error
	RequestCharacterSlotIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error
	RequestPurchase(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error
	MoveFromCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error
	MoveToCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) Enter(characterId uint32, m _map.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(CharacterEnterCashShopStatusEventProvider(characterId, m))
}

func (p *ProcessorImpl) Exit(characterId uint32, m _map.Model) error {
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

func (p *ProcessorImpl) RequestInventoryIncreasePurchaseByType(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error {
	p.l.Debugf("Character [%d] purchasing inventory [%d] expansion using currency [%d].", characterId, inventoryType, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByTypeCommandProvider(characterId, currency, inventoryType))
}

func (p *ProcessorImpl) RequestInventoryIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing inventory expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestStorageIncreasePurchase(characterId uint32, isPoints bool, currency uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion using currency [%d].", characterId, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseCommandProvider(characterId, currency))
}

func (p *ProcessorImpl) RequestStorageIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestCharacterSlotIncreasePurchaseByItem(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing character slot expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCharacterSlotIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestPurchase(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error {
	p.l.Debugf("Character [%d] purchasing [%d] with currency [%d], zero [%d]", characterId, serialNumber, currency, zero)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestPurchaseCommandProvider(characterId, serialNumber, currency))
}

func (p *ProcessorImpl) MoveFromCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error {
	p.l.Infof("Character [%d] moving [%d] to inventory [%d] to slot [%d].", characterId, serialNumber, inventoryType, slot)
	cp := compartment2.NewProcessor(p.l, p.ctx)

	// Get the character's destination compartment
	// TODO identify correct compartment type
	cscm, err := compartment.NewProcessor(p.l, p.ctx).GetByAccountIdAndType(accountId, compartment.TypeExplorer)
	if err != nil {
		return err
	}

	// Get the character's source compartment
	ccm, err := cp.GetByType(characterId, inventory.Type(inventoryType))
	if err != nil {
		return err
	}

	// Check if the compartment has the asset with CashId matching the serialNumber
	var foundAsset *asset2.Model
	for _, a := range cscm.Assets() {
		// Check if the asset is cash-equipable, cash, or pet
		if uint64(a.Item().CashId()) == serialNumber {
			assetCopy := a
			foundAsset = &assetCopy
			break
		}
	}

	if foundAsset == nil {
		p.l.Errorf("Character [%d] does not have asset with CashId [%d] in inventory [%d].", characterId, serialNumber, inventoryType)
		return errors.New("asset not found")
	}
	return cp.Transfer(accountId, characterId, foundAsset.Item().Id(), cscm.Id(), byte(cscm.Type()), "CASH_SHOP", ccm.Id(), byte(ccm.Type()), "CHARACTER", foundAsset.Item().Id())
}

func (p *ProcessorImpl) MoveToCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte) error {
	p.l.Infof("Character [%d] moving [%d] from inventory [%d] to cash inventory.", characterId, serialNumber, inventoryType)
	cp := compartment2.NewProcessor(p.l, p.ctx)

	// Get the character's destination compartment
	// TODO identify correct compartment type
	cscm, err := compartment.NewProcessor(p.l, p.ctx).GetByAccountIdAndType(accountId, compartment.TypeExplorer)
	if err != nil {
		return err
	}

	// Get the character's source compartment
	ccm, err := cp.GetByType(characterId, inventory.Type(inventoryType))
	if err != nil {
		return err
	}

	// Check if the compartment has the asset with CashId matching the serialNumber
	var foundAsset *asset.Model[any]
	for _, a := range ccm.Assets() {
		// Check if the asset is cash-equipable, cash, or pet
		if a.IsCashEquipable() {
			if cashEquipable, ok := a.ReferenceData().(asset.CashEquipableReferenceData); ok {
				if uint64(cashEquipable.CashId()) == serialNumber {
					assetCopy := a
					foundAsset = &assetCopy
					break
				}
			}
		} else if a.IsCash() {
			if cash, ok := a.ReferenceData().(asset.CashReferenceData); ok {
				if uint64(cash.CashId()) == serialNumber {
					assetCopy := a
					foundAsset = &assetCopy
					break
				}
			}
		} else if a.IsPet() {
			if pet, ok := a.ReferenceData().(asset.PetReferenceData); ok {
				if uint64(pet.CashId()) == serialNumber {
					assetCopy := a
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
	return cp.Transfer(accountId, characterId, foundAsset.Id(), ccm.Id(), byte(ccm.Type()), "CHARACTER", cscm.Id(), byte(cscm.Type()), "CASH_SHOP", foundAsset.ReferenceId())
}
