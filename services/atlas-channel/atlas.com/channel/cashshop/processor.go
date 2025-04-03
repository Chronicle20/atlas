package cashshop

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, m _map.Model) error {
	return func(ctx context.Context) func(characterId uint32, m _map.Model) error {
		return func(characterId uint32, m _map.Model) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterEnterCashShopStatusEventProvider(characterId, m))
		}
	}
}

func Exit(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, m _map.Model) error {
	return func(ctx context.Context) func(characterId uint32, m _map.Model) error {
		return func(characterId uint32, m _map.Model) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterExitCashShopStatusEventProvider(characterId, m))
		}
	}
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

func RequestInventoryIncreasePurchaseByType(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error {
	return func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error {
		return func(characterId uint32, isPoints bool, currency uint32, inventoryType byte) error {
			l.Debugf("Character [%d] purchasing inventory [%d] expansion using currency [%d].", characterId, inventoryType, currency)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestInventoryIncreaseByTypeCommandProvider(characterId, currency, inventoryType))
		}
	}
}

func RequestInventoryIncreasePurchaseByItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	return func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
		return func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
			l.Debugf("Character [%d] purchasing inventory expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestInventoryIncreaseByItemCommandProvider(characterId, currency, serialNumber))
		}
	}
}

func RequestStorageIncreasePurchase(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32) error {
	return func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32) error {
		return func(characterId uint32, isPoints bool, currency uint32) error {
			l.Debugf("Character [%d] purchasing storage expansion using currency [%d].", characterId, currency)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestStorageIncreaseCommandProvider(characterId, currency))
		}
	}
}

func RequestStorageIncreasePurchaseByItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	return func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
		return func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
			l.Debugf("Character [%d] purchasing storage expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestStorageIncreaseByItemCommandProvider(characterId, currency, serialNumber))
		}
	}
}

func RequestCharacterSlotIncreasePurchaseByItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
	return func(ctx context.Context) func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
		return func(characterId uint32, isPoints bool, currency uint32, serialNumber uint32) error {
			l.Debugf("Character [%d] purchasing character slot expansion via item [%d] using currency [%d]", characterId, serialNumber, currency)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestCharacterSlotIncreaseByItemCommandProvider(characterId, currency, serialNumber))
		}
	}
}

func RequestPurchase(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error {
	return func(ctx context.Context) func(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error {
		return func(characterId uint32, serialNumber uint32, isPoints bool, currency uint32, zero uint32) error {
			l.Debugf("Character [%d] purchasing [%d] with currency [%d], zero [%d]", characterId, serialNumber, currency, zero)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(RequestPurchaseCommandProvider(characterId, serialNumber, currency))
		}
	}
}
