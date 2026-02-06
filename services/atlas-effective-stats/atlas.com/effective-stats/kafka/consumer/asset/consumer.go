package asset

import (
	"atlas-effective-stats/character"
	"atlas-effective-stats/external/inventory"
	consumer2 "atlas-effective-stats/kafka/consumer"
	"atlas-effective-stats/kafka/message/asset"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoved)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeleted)))
	}
}

func handleAssetMoved(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeMoved {
		return
	}

	// Only handle equipable items
	if !asset.IsEquipable(e.Body.ReferenceType) {
		return
	}

	oldSlot := e.Body.OldSlot
	newSlot := e.Slot

	l.Debugf("Processing asset moved event for character [%d], asset [%d], template [%d], from slot [%d] to slot [%d].",
		e.CharacterId, e.AssetId, e.TemplateId, oldSlot, newSlot)

	if asset.IsEquipAction(oldSlot, newSlot) {
		// Item was equipped - need to add stat bonuses
		// For now, we fetch the equipment stats from inventory service
		handleItemEquipped(l, ctx, e)
	} else if asset.IsUnequipAction(oldSlot, newSlot) {
		// Item was unequipped - remove stat bonuses
		handleItemUnequipped(l, ctx, e)
	}
}

func handleItemEquipped(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	l.Debugf("Equipment [%d] equipped by character [%d], fetching stats.", e.AssetId, e.CharacterId)

	// Fetch equipment stats from inventory service
	compartment, err := inventory.RequestEquipCompartment(e.CharacterId)(l, ctx)
	if err != nil {
		l.WithError(err).Errorf("Failed to fetch equipment data for character [%d].", e.CharacterId)
		return
	}

	// Find the asset we just equipped
	var equipData *inventory.EquipableRestData
	for _, a := range compartment.Assets {
		if a.Id == e.AssetId {
			data, ok := a.GetEquipableData()
			if ok {
				equipData = &data
			}
			break
		}
	}

	if equipData == nil {
		l.Warnf("Could not find equipment data for asset [%d].", e.AssetId)
		return
	}

	// Convert equipment stats to bonuses
	bonuses := extractEquipmentBonuses(e.AssetId, equipData)
	if len(bonuses) > 0 {
		// We need worldId and channelId - get them from the registry or use defaults
		// For now, use 0 as we'll update the existing entry
		ch := channel.NewModel(0, 0)
		if err := character.NewProcessor(l, ctx).AddEquipmentBonuses(ch, e.CharacterId, e.AssetId, bonuses); err != nil {
			l.WithError(err).Errorf("Failed to add equipment bonuses for character [%d].", e.CharacterId)
		}
	}
}

func handleItemUnequipped(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	l.Debugf("Equipment [%d] unequipped by character [%d].", e.AssetId, e.CharacterId)

	// Remove all stat bonuses from this equipment
	if err := character.NewProcessor(l, ctx).RemoveEquipmentBonuses(e.CharacterId, e.AssetId); err != nil {
		l.WithError(err).Errorf("Failed to remove equipment bonuses for character [%d].", e.CharacterId)
	}
}

func handleAssetDeleted(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.DeletedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeDeleted {
		return
	}

	// Only handle if it was equipped (negative slot)
	if !asset.IsEquipmentSlot(e.Slot) {
		return
	}

	l.Debugf("Equipped asset [%d] deleted for character [%d].", e.AssetId, e.CharacterId)

	p := character.NewProcessor(l, ctx)

	// Remove equipment bonuses
	if err := p.RemoveEquipmentBonuses(e.CharacterId, e.AssetId); err != nil {
		l.WithError(err).Debugf("Failed to remove equipment bonuses (may not exist).")
	}
}

func extractEquipmentBonuses(assetId uint32, equipData *inventory.EquipableRestData) []stat.Bonus {
	bonuses := make([]stat.Bonus, 0)
	source := fmt.Sprintf("equipment:%d", assetId)

	if equipData.Strength > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeStrength, int32(equipData.Strength)))
	}
	if equipData.Dexterity > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeDexterity, int32(equipData.Dexterity)))
	}
	if equipData.Luck > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeLuck, int32(equipData.Luck)))
	}
	if equipData.Intelligence > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeIntelligence, int32(equipData.Intelligence)))
	}
	if equipData.Hp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHP, int32(equipData.Hp)))
	}
	if equipData.Mp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMP, int32(equipData.Mp)))
	}
	if equipData.WeaponAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(equipData.WeaponAttack)))
	}
	if equipData.MagicAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(equipData.MagicAttack)))
	}
	if equipData.WeaponDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(equipData.WeaponDefense)))
	}
	if equipData.MagicDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(equipData.MagicDefense)))
	}
	if equipData.Accuracy > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(equipData.Accuracy)))
	}
	if equipData.Avoidability > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(equipData.Avoidability)))
	}
	if equipData.Speed > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(equipData.Speed)))
	}
	if equipData.Jump > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(equipData.Jump)))
	}

	return bonuses
}
