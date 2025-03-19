package consumable

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func RequestItemConsume(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, updateTime uint32) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, updateTime uint32) error {
		return func(characterId uint32, itemId uint32, slot int16, updateTime uint32) error {
			l.Debugf("Character [%d] using item [%d] from slot [%d]. updateTime [%d]", characterId, itemId, slot, updateTime)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestItemConsumeCommandProvider(characterId, slot, itemId, 1))
		}
	}
}

func RequestScrollUse(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
	return func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
		return func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool, updateTime uint32) error {
			l.Debugf("Character [%d] attempting to scroll item in slot [%d] with scroll from slot [%d]. whiteScroll [%t], legendarySpirit [%t], updateTime [%d].", characterId, equipSlot, scrollSlot, whiteScroll, legendarySpirit, updateTime)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestScrollCommandProvider(characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit))
		}
	}
}
