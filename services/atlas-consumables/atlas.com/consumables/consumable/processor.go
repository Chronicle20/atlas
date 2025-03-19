package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/character/equipment/slot"
	"atlas-consumables/character/inventory/item"
	"atlas-consumables/inventory"
	inventory3 "atlas-consumables/kafka/message/inventory"
	once "atlas-consumables/kafka/once/inventory"
	"atlas-consumables/map"
	cim "atlas-consumables/map/character"
	"atlas-consumables/map/data"
	"atlas-consumables/pet"
	"context"
	"errors"
	ts "github.com/Chronicle20/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas-constants/item"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"math"
)

type ItemConsumer func(l logrus.FieldLogger) func(ctx context.Context) error

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) (Model, error) {
	return func(ctx context.Context) func(itemId uint32) (Model, error) {
		return func(itemId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(itemId), Extract)()
		}
	}
}

const (
	WhiteScrollOnePercent     = uint32(2049000)
	WhiteScrollThreePerecent  = uint32(2049001)
	WhiteScrollFivePerecent   = uint32(2049002)
	WhiteScrollTwentyPerecent = uint32(2049003)
)

func RequestItemConsume(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, slot int16, itemId item2.Id, quantity int16) error {
	return func(ctx context.Context) func(characterId uint32, slot int16, itemId item2.Id, quantity int16) error {
		return func(characterId uint32, slot int16, itemId item2.Id, quantity int16) error {
			transactionId := uuid.New()
			l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
			t, _ := topic.EnvProvider(l)(inventory3.EnvEventInventoryChanged)()
			validator := once.ReservationValidator(transactionId)

			var itemConsumer ItemConsumer

			if item2.GetClassification(itemId) == item2.Classification(200) || item2.GetClassification(itemId) == item2.Classification(201) || item2.GetClassification(itemId) == item2.Classification(202) {
				itemConsumer = ConsumeStandard(transactionId, characterId, slot, itemId, quantity)
			} else if item2.GetClassification(itemId) == item2.ClassificationConsumableTownWarp {
				itemConsumer = ConsumeTownScroll(transactionId, characterId, slot, itemId, quantity)
			} else if item2.GetClassification(itemId) == item2.ClassificationConsumablePetFood {
				itemConsumer = ConsumePetFood(transactionId, characterId, slot, itemId, quantity)
			}

			handler := inventory.Consume(itemConsumer)

			_, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

			err = inventory.RequestReserve(l)(ctx)(transactionId, characterId, inventory2.TypeValueUse, []inventory.Reserves{{Slot: slot, ItemId: uint32(itemId), Quantity: quantity}})
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func RequestScroll(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
	return func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
		return func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
			c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				return err
			}

			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				return err
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				return errors.New("equipment not found")
			}
			if sm.Equipable.Slots() <= 0 {
				return errors.New("equipment slots is zero")
			}

			reserves := make([]inventory.Reserves, 0)
			scrollItem, ok := c.Inventory().Use().FindBySlot(scrollSlot)
			if !ok {
				return errors.New("scroll item not found")
			}

			reserves = append(reserves, inventory.Reserves{
				Slot:     scrollSlot,
				ItemId:   scrollItem.ItemId(),
				Quantity: 1,
			})

			var whiteScrollItem *item.Model
			if whiteScroll {
				// TODO figure out which one to use?
				found := false
				for _, i := range c.Inventory().Use().Items() {
					if i.ItemId() == WhiteScrollOnePercent || i.ItemId() == WhiteScrollThreePerecent || i.ItemId() == WhiteScrollFivePerecent || i.ItemId() == WhiteScrollTwentyPerecent {
						whiteScrollItem = &i
						found = true
					}
				}
				if !found {
					return errors.New("white scroll item not found")
				}
				reserves = append(reserves, inventory.Reserves{
					Slot:     whiteScrollItem.Slot(),
					ItemId:   whiteScrollItem.ItemId(),
					Quantity: 1,
				})
			}

			transactionId := uuid.New()
			l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
			t, _ := topic.EnvProvider(l)(inventory3.EnvEventInventoryChanged)()
			validator := once.ReservationValidator(transactionId)
			handler := inventory.Consume(ConsumeScroll(transactionId, characterId, &scrollItem, equipSlot, whiteScrollItem, legendarySpirit))
			_, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

			err = inventory.RequestReserve(l)(ctx)(transactionId, characterId, inventory2.TypeValueUse, reserves)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func ConsumeScroll(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, equipSlot int16, whiteScrollItem *item.Model, legendarySpirit bool) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			l.Debugf("Character [%d] has reserved items. Consume scroll in slot [%d].", characterId, scrollItem.Slot())
			c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem, whiteScrollItem)
				return err
			}

			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem, whiteScrollItem)
				return err
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem, whiteScrollItem)
				return errors.New("equipment not found")
			}
			if sm.Equipable.Slots() <= 0 {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem, whiteScrollItem)
				return errors.New("equipment slots is zero")
			}

			// TODO perform scroll

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.ItemId(), characterId)
			}
			if whiteScrollItem != nil {
				err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, whiteScrollItem.Slot())
				if err != nil {
					l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", whiteScrollItem.ItemId(), characterId)
				}
			}

			// TODO emit scrolling event
			return nil
		}
	}
}

func CancelScroll(l logrus.FieldLogger) func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, whiteScrollItem *item.Model) error {
	return func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, whiteScrollItem *item.Model) error {
		return func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, whiteScrollItem *item.Model) error {
			_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			if whiteScrollItem != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, whiteScrollItem.Slot())
			}
			return nil
		}
	}
}

func ConsumeStandard(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, quantity int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			c, err := character.GetById(l)(ctx)()(characterId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			m, err := cim.GetMap(characterId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			statups := make([]stat.Model, 0)
			duration := int32(0)
			if val, ok := ci.GetSpec(SpecTypeAccuracy); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeAccuracy,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeEvasion); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeAvoidability,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeHP); ok && val > 0 {
				_ = character.ChangeHP(l)(ctx)(m, characterId, int16(val))
			}
			if val, ok := ci.GetSpec(SpecTypeHPRecovery); ok && val > 0 {
				pct := float64(val) / float64(100)
				res := int16(math.Floor(float64(c.MaxHp()) * pct))
				_ = character.ChangeHP(l)(ctx)(m, characterId, res)
			}
			if val, ok := ci.GetSpec(SpecTypeJump); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeJump,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMagicAttack); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeMagicAttack,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMagicDefense); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeMagicDefense,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMP); ok && val > 0 {
				_ = character.ChangeMP(l)(ctx)(m, characterId, int16(val))
			}
			if val, ok := ci.GetSpec(SpecTypeMPRecovery); ok && val > 0 {
				pct := float64(val) / float64(100)
				res := int16(math.Floor(float64(c.MaxMp()) * pct))
				_ = character.ChangeMP(l)(ctx)(m, characterId, res)
			}
			if val, ok := ci.GetSpec(SpecTypeWeaponAttack); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeWeaponAttack,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeWeaponDefense); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeWeaponDefense,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeSpeed); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeSpeed,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeTime); ok && val > 0 {
				duration = val / 1000
			}

			if len(statups) > 0 {
				_ = buff.Apply(l)(ctx)(m, characterId, -int32(itemId), duration, statups)(characterId)
			}
			return nil
		}
	}
}

func ConsumeTownScroll(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, quantity int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			m, err := cim.GetMap(characterId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}
			toMapId := _map2.EmptyMapId
			if val, ok := ci.GetSpec(SpecTypeMoveTo); ok && val > 0 {
				toMapId = _map2.Id(val)
			}
			if toMapId == _map2.EmptyMapId {
				mm, err := data.GetById(l)(ctx)(m.MapId())
				if err != nil {
					_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
					return err
				}
				toMapId = _map2.Id(mm.ReturnMapId())
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			err = _map.WarpRandom(l)(ctx)(_map2.NewModel(m.WorldId())(m.ChannelId())(toMapId))(characterId)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func ConsumePetFood(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, quantity int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p, err := pet.HungriestByOwnerProvider(l)(ctx)(characterId)()
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}
			inc := byte(0)
			if val, ok := ci.GetSpec(SpecTypeInc); ok {
				inc = byte(val)
			}

			err = pet.AwardFullness(l)(ctx)(characterId, p.Id(), inc)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}
			return nil
		}
	}
}
