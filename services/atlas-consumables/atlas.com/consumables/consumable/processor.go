package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/character/equipment/slot"
	"atlas-consumables/character/inventory/item"
	"atlas-consumables/equipable"
	"atlas-consumables/inventory"
	"atlas-consumables/kafka/message/consumable"
	inventory3 "atlas-consumables/kafka/message/inventory"
	once "atlas-consumables/kafka/once/inventory"
	"atlas-consumables/kafka/producer"
	consumable2 "atlas-consumables/kafka/producer/consumable"
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
	"math/rand"
)

type ItemConsumer func(l logrus.FieldLogger) func(ctx context.Context) error

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) (Model, error) {
	return func(ctx context.Context) func(itemId uint32) (Model, error) {
		return func(itemId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(itemId), Extract)()
		}
	}
}

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

			if whiteScroll && !item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) {
				return errors.New("using but not using white scroll")
			}

			transactionId := uuid.New()
			l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
			t, _ := topic.EnvProvider(l)(inventory3.EnvEventInventoryChanged)()
			validator := once.ReservationValidator(transactionId)
			handler := inventory.Consume(ConsumeScroll(transactionId, characterId, &scrollItem, equipSlot, whiteScroll, legendarySpirit))
			_, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

			err = inventory.RequestReserve(l)(ctx)(transactionId, characterId, inventory2.TypeValueUse, reserves)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func ConsumeScroll(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, equipSlot int16, whiteScroll bool, legendarySpirit bool) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			l.Debugf("Character [%d] has reserved items. Consume scroll in slot [%d].", characterId, scrollItem.Slot())
			c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
				return err
			}

			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
				return err
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
				return errors.New("equipment not found")
			}
			if sm.Equipable.Slots() <= 0 {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
				return errors.New("equipment slots is zero")
			}

			ci, err := GetById(l)(ctx)(scrollItem.ItemId())
			if err != nil {
				_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
				return err
			}

			// TODO consume vega scroll
			successProb := ci.success

			// TODO spikes / cursed property
			isSuccess := rand.Int31n(100) <= int32(successProb)

			isCursed := false
			if isSuccess {
				l.Debugf("Character [%d] has passed scroll [%d].", characterId, scrollItem.ItemId())
				if item2.IsScrollSpikes(item2.Id(scrollItem.ItemId())) {
					// TODO do spike property
				} else if item2.IsScrollColdProtection(item2.Id(scrollItem.ItemId())) {
					// TODO do cold property
				} else if item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) {
					err = equipable.ChangeStat(l)(ctx)(characterId, sm.Equipable.ItemId(), sm.Equipable.Slot(), equipable.AddSlots(1))
					if err != nil {
						_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
						return err
					}
				} else if item2.IsChaosScroll(item2.Id(scrollItem.ItemId())) {
					// TODO apply chaos
				} else {
					err = equipable.ChangeStat(l)(ctx)(characterId, sm.Equipable.ItemId(), sm.Equipable.Slot(),
						equipable.AddStrength(int16(ci.incSTR)),
						equipable.AddDexterity(int16(ci.incDEX)),
						equipable.AddIntelligence(int16(ci.incINT)),
						equipable.AddLuck(int16(ci.incLUK)),
						equipable.AddHP(int16(ci.incMHP)),
						equipable.AddMP(int16(ci.incMMP)),
						equipable.AddWeaponAttack(int16(ci.incPAD)),
						equipable.AddMagicAttack(int16(ci.incMAD)),
						equipable.AddWeaponDefense(int16(ci.incPDD)),
						equipable.AddMagicDefense(int16(ci.incMDD)),
						equipable.AddAccuracy(int16(ci.incACC)),
						equipable.AddAvoidability(int16(ci.incEVA)),
						equipable.AddHands(0),
						equipable.AddSpeed(int16(ci.incSpeed)),
						equipable.AddJump(int16(ci.incJump)),
						equipable.AddSlots(-1),
						equipable.AddLevel(1),
					)
					if err != nil {
						_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
						return err
					}
				}
			} else {
				l.Debugf("Character [%d] has failed scroll [%d].", characterId, scrollItem.ItemId())
				if !item2.IsScrollSpikes(item2.Id(scrollItem.ItemId())) && !item2.IsScrollColdProtection(item2.Id(scrollItem.ItemId())) && !item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) {
					err = equipable.ChangeStat(l)(ctx)(characterId, sm.Equipable.ItemId(), sm.Equipable.Slot(), equipable.AddSlots(-1))
					if err != nil {
						_ = CancelScroll(l)(ctx)(transactionId, characterId, scrollItem)
						return err
					}
				}
				if rand.Int31n(100) <= int32(ci.cursed) {
					l.Debugf("Character [%d] item has been cursed.", characterId)
					isCursed = true
				}
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.ItemId(), characterId)
			}

			if isCursed {
				err = inventory.DestroyItem(l)(ctx)(characterId, inventory2.TypeValueEquip, equipSlot)
				if err != nil {
					l.WithError(err).Errorf("Unable to destroy item in slot [%d] for character [%d] during scrolling.", equipSlot, characterId)
				}
			}

			if isSuccess {
				_ = PassScroll(l)(ctx)(characterId, legendarySpirit, whiteScroll)
			} else {
				_ = FailScroll(l)(ctx)(characterId, isCursed, legendarySpirit, whiteScroll)
			}
			return nil
		}
	}
}

func CancelScroll(l logrus.FieldLogger) func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model) error {
	return func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model) error {
		return func(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model) error {
			_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			return nil
		}
	}
}

func PassScroll(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, legendarySpirit bool, whiteScroll bool) error {
	return func(ctx context.Context) func(characterId uint32, legendarySpirit bool, whiteScroll bool) error {
		return func(characterId uint32, legendarySpirit bool, whiteScroll bool) error {
			return producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(consumable2.ScrollEventProvider(characterId)(true, false, legendarySpirit, whiteScroll))
		}
	}
}

func FailScroll(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error {
	return func(ctx context.Context) func(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error {
		return func(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error {
			return producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(consumable2.ScrollEventProvider(characterId)(false, cursed, legendarySpirit, whiteScroll))
		}
	}
}
