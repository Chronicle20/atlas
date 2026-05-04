package asset

import (
	"atlas-effective-stats/character"
	"atlas-effective-stats/external/inventory"
	consumer2 "atlas-effective-stats/kafka/consumer"
	"atlas-effective-stats/kafka/message/asset"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoved))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeleted))); err != nil {
			return err
		}
		return nil
	}
}

func handleAssetMoved(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeMoved {
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
	l.Debugf("Equipment [%d] (template %d) equipped by character [%d], fetching stats.", e.AssetId, e.TemplateId, e.CharacterId)

	compartment, err := inventory.RequestEquipCompartment(e.CharacterId)(l, ctx)
	if err != nil {
		l.WithError(err).Errorf("Failed to fetch equipment data for character [%d].", e.CharacterId)
		return
	}

	var (
		assetModel     *inventory.AssetRestModel
		foundInCompart bool
		actualSlot     int16
	)
	for _, a := range compartment.Assets {
		if a.Id == e.AssetId {
			foundInCompart = true
			actualSlot = a.Slot
			am := a
			assetModel = &am
			break
		}
	}

	if assetModel == nil || !assetModel.IsEquipped() {
		if !foundInCompart {
			l.Warnf("Equip event for asset [%d] character [%d] but asset is not present in the equip compartment (event slot=[%d] oldSlot=[%d], compartment has %d assets). Likely a read-after-write race or concurrent delete.",
				e.AssetId, e.CharacterId, e.Slot, e.Body.OldSlot, len(compartment.Assets))
		} else {
			l.Warnf("Equip event for asset [%d] character [%d] but inventory shows asset at non-equipped slot [%d] (event claims slot=[%d] oldSlot=[%d]). Possible upstream MOVED slot-semantic inversion — verify atlas-inventory's MovedEventStatusProvider call site.",
				e.AssetId, e.CharacterId, actualSlot, e.Slot, e.Body.OldSlot)
		}
		return
	}

	bonuses := extractEquipmentBonuses(e.AssetId, *assetModel)
	ch := channel.NewModel(0, 0)
	if err := character.NewProcessor(l, ctx).AddEquipmentBonuses(ch, e.CharacterId, e.AssetId, e.TemplateId, bonuses); err != nil {
		l.WithError(err).Errorf("Failed to add equipment bonuses for character [%d].", e.CharacterId)
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

func extractEquipmentBonuses(assetId uint32, a inventory.AssetRestModel) []stat.Bonus {
	bonuses := make([]stat.Bonus, 0)
	source := fmt.Sprintf("equipment:%d", assetId)

	if a.Strength > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeStrength, int32(a.Strength)))
	}
	if a.Dexterity > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeDexterity, int32(a.Dexterity)))
	}
	if a.Luck > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeLuck, int32(a.Luck)))
	}
	if a.Intelligence > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeIntelligence, int32(a.Intelligence)))
	}
	if a.Hp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHp, int32(a.Hp)))
	}
	if a.Mp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMp, int32(a.Mp)))
	}
	if a.WeaponAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(a.WeaponAttack)))
	}
	if a.MagicAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(a.MagicAttack)))
	}
	if a.WeaponDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(a.WeaponDefense)))
	}
	if a.MagicDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(a.MagicDefense)))
	}
	if a.Accuracy > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(a.Accuracy)))
	}
	if a.Avoidability > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(a.Avoidability)))
	}
	if a.Speed > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(a.Speed)))
	}
	if a.Jump > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(a.Jump)))
	}

	return bonuses
}
