package cashshop

import (
	"atlas-channel/kafka/message/cashshop"
	"atlas-channel/kafka/producer"
	"context"
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
