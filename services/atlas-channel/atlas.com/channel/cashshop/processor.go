package cashshop

import (
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/kafka/message/cashshop"
	"atlas-channel/saga"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
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
	RequestPurchase(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32, transactionId uuid.UUID, operation string) error
	RequestCouponRedemption(characterId uint32, code string) error
	MoveFromCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error
	MoveToCashInventory(accountId uint32, characterId uint32, serialNumber uint64, inventoryType byte) error
	OpenSurprise(accountId uint32, characterId uint32, cashId int64) error
	RequestLockerRebate(accountId uint32, characterId uint32, cashId int64, transactionId uuid.UUID) error
	RequestGiftPurchase(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, message string) error
	RequestPackagePurchase(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) error
	RequestRingPurchase(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, message string, ringType string) error
	RequestEquipSlotIncrease(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32) error
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

var _ Processor = (*ProcessorImpl)(nil)

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

func (p *ProcessorImpl) RequestPurchase(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32, transactionId uuid.UUID, operation string) error {
	currency = resolvePurchaseCurrency(isPoints, currency)
	p.l.Debugf("Character [%d] purchasing [%d] with currency [%d], zero [%d], transaction [%s], operation [%s]", characterId, serialNumber, currency, zero, transactionId, operation)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestPurchaseCommandProvider(characterId, serialNumber, currency, transactionId, operation))
}

// RequestCouponRedemption forwards an already-normalized coupon code to
// atlas-cashshop. The code must NEVER be logged: it is a redeemable bearer
// token, so only its length goes into the log line.
func (p *ProcessorImpl) RequestCouponRedemption(characterId uint32, code string) error {
	p.l.Debugf("Character [%d] submitting a coupon code of length [%d].", characterId, len(code))
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCouponRedemptionCommandProvider(characterId, code))
}

// resolvePurchaseCurrency maps the buy packet's isPoints flag onto the wallet
// currency code when no currency was provided on the wire. JMS cash buys carry
// isPoints but no currency (currency==0), so an isPoints buy must be steered to
// the MaplePoints balance (code 2) instead of falling through to prepaid. GMS
// sends an explicit non-zero currency, so this guard never fires for GMS and
// leaves its behavior unchanged.
func resolvePurchaseCurrency(isPoints bool, currency uint32) uint32 {
	const walletCurrencyMaplePoints = uint32(2) // wallet.Model.Purchase: currency==2 -> points
	if isPoints && currency == 0 {
		return walletCurrencyMaplePoints
	}
	return currency
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
	step := saga.Step{
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
		Steps:         []saga.Step{step},
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
	step := saga.Step{
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
		Steps:         []saga.Step{step},
	}

	err := sagaP.Create(sagaTx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to create saga for transferring cash item [%d] to cash shop for character [%d].", serialNumber, characterId)
		return err
	}

	p.l.Debugf("Created transfer saga [%s] for character [%d] transferring cash item [%d] to cash shop.", transactionId.String(), characterId, serialNumber)
	return nil
}

// OpenSurprise forwards a Cash Shop Surprise open request. The transaction
// id is minted here, once per click: atlas-cashshop's openings ledger keys
// idempotency on it, so a Kafka redelivery replays this id and is rejected
// while a genuine second click gets a fresh one.
func (p *ProcessorImpl) OpenSurprise(accountId uint32, characterId uint32, cashId int64) error {
	transactionId := uuid.New()
	p.l.Debugf("Character [%d] opening surprise box [%d]. Transaction [%s].", characterId, cashId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(OpenSurpriseCommandProvider(characterId, transactionId, accountId, cashId))
}

// RequestLockerRebate forwards a locker item REBATE request. TransactionId is
// minted by the caller (once per click, mirroring OpenSurprise's idempotency
// pattern) so a Kafka redelivery replays this id and is rejected by
// atlas-cashshop's rebate ledger while a genuine second click gets a fresh
// one.
func (p *ProcessorImpl) RequestLockerRebate(accountId uint32, characterId uint32, cashId int64, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] requesting locker rebate for cash item [%d]. Transaction [%s].", characterId, cashId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestLockerRebateCommandProvider(characterId, transactionId, accountId, cashId))
}

// RequestGiftPurchase forwards a GIFT purchase request. TransactionId is
// minted by the caller (once per click, mirroring OpenSurprise/RequestLockerRebate's
// idempotency pattern) so a Kafka redelivery replays this id and is rejected
// by atlas-cashshop's gift ledger while a genuine second click gets a fresh
// one.
func (p *ProcessorImpl) RequestGiftPurchase(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, message string) error {
	p.l.Debugf("Character [%d] gifting serial [%d] to character [%d]. Transaction [%s].", characterId, serialNumber, recipientCharacterId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestGiftPurchaseCommandProvider(characterId, transactionId, serialNumber, recipientCharacterId, senderName, message))
}

// RequestPackagePurchase forwards a CASH PACKAGE purchase request (task-240
// task 17, atlas-channel half of task 16's REQUEST_PACKAGE_PURCHASE
// command). TransactionId is minted by the caller (once per click, mirroring
// OpenSurprise/RequestGiftPurchase's idempotency pattern) so a Kafka
// redelivery replays this id and is rejected by atlas-cashshop's package
// ledger while a genuine second click gets a fresh one.
// RecipientCharacterId == 0 means buy-for-self (BUY_PACKAGE, mode 30/32);
// non-zero means gift (BUY_OTHER_PACKAGE, mode 31/33) -- the same single
// command shape task 16 built on the atlas-cashshop side.
//
// isPoints/currency go through resolvePurchaseCurrency exactly like
// RequestPurchase above -- the same helper, the same call shape. For
// BUY_PACKAGE (handleBuyPackage), that resolution is load-bearing: it maps
// the wire's pointType bool onto a currency code. For BUY_OTHER_PACKAGE
// (handleBuyOtherPackage), the caller always passes isPoints=false with an
// already-final currency (walletCurrencyPrepaid, 3), so
// resolvePurchaseCurrency's only branch (isPoints && currency==0) can never
// fire and the call is an inert passthrough -- see handleBuyOtherPackage's
// own doc comment for why that caller does not depend on this resolution.
func (p *ProcessorImpl) RequestPackagePurchase(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) error {
	currency = resolvePurchaseCurrency(isPoints, currency)
	p.l.Debugf("Character [%d] purchasing package serial [%d] with currency [%d] for recipient [%d]. Transaction [%s].", characterId, serialNumber, currency, recipientCharacterId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestPackagePurchaseCommandProvider(characterId, transactionId, currency, serialNumber, recipientCharacterId, senderName))
}

// RequestRingPurchase forwards a RING pair purchase request (BUY_COUPLE /
// BUY_FRIENDSHIP, task-240 task 20, atlas-channel half of task 19's
// REQUEST_RING_PURCHASE command). TransactionId is minted by the caller
// (once per click, mirroring RequestGiftPurchase/RequestPackagePurchase's
// idempotency pattern) so a Kafka redelivery replays this id and is
// rejected by atlas-cashshop's ring ledger while a genuine second click
// gets a fresh one. isPoints/currency go through resolvePurchaseCurrency
// exactly like RequestPurchase/RequestPackagePurchase above.
func (p *ProcessorImpl) RequestRingPurchase(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, message string, ringType string) error {
	currency = resolvePurchaseCurrency(isPoints, currency)
	p.l.Debugf("Character [%d] purchasing ring serial [%d] with currency [%d] for partner [%d]. Transaction [%s].", characterId, serialNumber, currency, partnerCharacterId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestRingPurchaseCommandProvider(characterId, transactionId, currency, serialNumber, partnerCharacterId, senderName, message, ringType))
}

// RequestEquipSlotIncrease forwards an ENABLE_EQUIP_SLOT purchase request
// (task-240 task 23, mode 9/10). TransactionId is minted by the caller
// (once per click, mirroring RequestPackagePurchase/RequestRingPurchase's
// idempotency pattern) so a Kafka redelivery replays this id and is
// rejected by atlas-cashshop's ledger while a genuine second click gets a
// fresh one. isPoints/currency go through resolvePurchaseCurrency exactly
// like every other purchase arm above.
func (p *ProcessorImpl) RequestEquipSlotIncrease(characterId uint32, transactionId uuid.UUID, isPoints bool, currency uint32, serialNumber uint32) error {
	currency = resolvePurchaseCurrency(isPoints, currency)
	p.l.Debugf("Character [%d] requesting equip slot increase serial [%d] with currency [%d]. Transaction [%s].", characterId, serialNumber, currency, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestEquipSlotIncreaseCommandProvider(characterId, transactionId, currency, serialNumber))
}
