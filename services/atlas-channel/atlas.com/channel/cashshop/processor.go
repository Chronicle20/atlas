package cashshop

import (
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/kafka/message/cashshop"
	"atlas-channel/kafka/producer"
	"atlas-channel/saga"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for cashshop processing
type Processor interface {
	Enter(characterId uint32, f field.Model) error
	Exit(characterId uint32, f field.Model) error
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

func (p *ProcessorImpl) Enter(characterId uint32, f field.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(CharacterEnterCashShopStatusEventProvider(characterId, f))
}

func (p *ProcessorImpl) Exit(characterId uint32, f field.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(CharacterExitCashShopStatusEventProvider(characterId, f))
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

func (p *ProcessorImpl) RequestInventoryIncreasePurchaseByType(characterId uint32, _ bool, currency uint32, inventoryType byte) error {
	p.l.Debugf("Character [%d] purchasing inventory [%d] expansion using currency [%d].", characterId, inventoryType, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByTypeCommandProvider(characterId, currency, inventoryType))
}

func (p *ProcessorImpl) RequestInventoryIncreasePurchaseByItem(characterId uint32, _ bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing inventory expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestInventoryIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestStorageIncreasePurchase(characterId uint32, _ bool, currency uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion using currency [%d].", characterId, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseCommandProvider(characterId, currency))
}

func (p *ProcessorImpl) RequestStorageIncreasePurchaseByItem(characterId uint32, _ bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing storage expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestStorageIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestCharacterSlotIncreasePurchaseByItem(characterId uint32, _ bool, currency uint32, serialNumber uint32) error {
	p.l.Debugf("Character [%d] purchasing character slot expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCharacterSlotIncreaseByItemCommandProvider(characterId, currency, serialNumber))
}

func (p *ProcessorImpl) RequestPurchase(characterId uint32, serialNumber uint32, _ bool, currency uint32, zero uint32) error {
	p.l.Debugf("Character [%d] purchasing [%d] with currency [%d], zero [%d]", characterId, serialNumber, currency, zero)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestPurchaseCommandProvider(characterId, serialNumber, currency))
}

func (p *ProcessorImpl) MoveFromCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte, _ int16) error {
	p.l.Infof("Character [%d] moving cash item [%d] to inventory [%d].", characterId, serialNumber, inventoryType)

	// Create saga transaction for withdrawing from cash shop
	sagaP := saga.NewProcessor(p.l, p.ctx)
	transactionId := uuid.New()
	now := time.Now()

	// TODO: identify correct compartment type based on character job
	compartmentType := byte(compartment.TypeExplorer)

	// Create the high-level withdrawal step (will be expanded by saga-orchestrator)
	step := saga.Step[any]{
		StepId: "withdraw_from_cash_shop",
		Status: saga.Pending,
		Action: saga.WithdrawFromCashShop,
		Payload: saga.WithdrawFromCashShopPayload{
			TransactionId:   transactionId,
			CharacterId:     characterId,
			AccountId:       accountId,
			CashId:          serialNumber,
			CompartmentType: compartmentType,
			InventoryType:   inventoryType,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sagaTx := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.CashShopOperation,
		InitiatedBy:   "CASH_SHOP",
		Steps:         []saga.Step[any]{step},
	}

	err := sagaP.Create(sagaTx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to create saga for withdrawing cash item [%d] for character [%d].", serialNumber, characterId)
		return err
	}

	p.l.Debugf("Created withdrawal saga [%s] for character [%d] withdrawing cash item [%d].", transactionId.String(), characterId, serialNumber)
	return nil
}

func (p *ProcessorImpl) MoveToCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte) error {
	p.l.Infof("Character [%d] moving cash item [%d] from inventory [%d] to cash inventory.", characterId, serialNumber, inventoryType)

	// Create saga transaction for transferring to cash shop
	sagaP := saga.NewProcessor(p.l, p.ctx)
	transactionId := uuid.New()
	now := time.Now()

	// TODO: identify correct compartment type based on character job
	compartmentType := byte(compartment.TypeExplorer)

	// Create the high-level transfer step (will be expanded by saga-orchestrator)
	step := saga.Step[any]{
		StepId: "transfer_to_cash_shop",
		Status: saga.Pending,
		Action: saga.TransferToCashShop,
		Payload: saga.TransferToCashShopPayload{
			TransactionId:       transactionId,
			CharacterId:         characterId,
			AccountId:           accountId,
			CashId:              serialNumber,
			SourceInventoryType: inventoryType,
			CompartmentType:     compartmentType,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sagaTx := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.CashShopOperation,
		InitiatedBy:   "CASH_SHOP",
		Steps:         []saga.Step[any]{step},
	}

	err := sagaP.Create(sagaTx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to create saga for transferring cash item [%d] to cash shop for character [%d].", serialNumber, characterId)
		return err
	}

	p.l.Debugf("Created transfer saga [%s] for character [%d] transferring cash item [%d] to cash shop.", transactionId.String(), characterId, serialNumber)
	return nil
}
