package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/character/equipment/slot"
	equipable2 "atlas-consumables/character/inventory/equipable"
	"atlas-consumables/character/inventory/item"
	"atlas-consumables/equipable"
	"atlas-consumables/equipable/statistic"
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
			validator := once.ReservationValidator(transactionId, uint32(itemId))

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

func ConsumeError(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error {
	return func(ctx context.Context) func(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error {
		return func(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error {
			l.Debugf("Character [%d] unable to consume item due to error: [%v]", characterId, err)
			cErr := inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if cErr != nil {
				l.WithError(cErr).Errorf("Unable to cancel item reservation at inventory [%d] slot [%d] for character [%d] as part of transaction [%d].", inventoryType, slot, characterId, transactionId)
			}
			cErr = producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(consumable2.ErrorEventProvider(characterId))
			if cErr != nil {
				l.WithError(cErr).Errorf("Unable to issue consumption error [%v] on event topic. Character [%d] likely going to be stuck.", err, characterId)
			}
			return err
		}
	}
}

func ConsumeStandard(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, quantity int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			c, err := character.GetById(l)(ctx)()(characterId)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			m, err := cim.GetMap(characterId)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
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
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			toMapId := _map2.EmptyMapId
			if val, ok := ci.GetSpec(SpecTypeMoveTo); ok && val > 0 {
				toMapId = _map2.Id(val)
			}
			if toMapId == _map2.EmptyMapId {
				mm, err := data.GetById(l)(ctx)(m.MapId())
				if err != nil {
					return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
				}
				toMapId = _map2.Id(mm.ReturnMapId())
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
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
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := GetById(l)(ctx)(uint32(itemId))
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			inc := byte(0)
			if val, ok := ci.GetSpec(SpecTypeInc); ok {
				inc = byte(val)
			}

			err = pet.AwardFullness(l)(ctx)(characterId, p.Id(), inc)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

func RequestScroll(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
	return func(ctx context.Context) func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
		return func(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
			transactionId := uuid.New()
			reserves := make([]inventory.Reserves, 0)

			c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, err)
			}

			scrollItem, ok := c.Inventory().Use().FindBySlot(scrollSlot)
			if !ok {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("scroll item not found"))
			}

			// Ensure if we are using a "white scroll" that we have a white scroll item.
			var whiteScrollItem *item.Model
			if whiteScroll {
				whiteScrollItem, ok = c.Inventory().Use().FindFirstByItemId(2340000)
				if !ok {
					return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("does not have white scroll"))
				}
				reserves = append(reserves, inventory.Reserves{
					Slot:     whiteScrollItem.Slot(),
					ItemId:   whiteScrollItem.ItemId(),
					Quantity: 1,
				})
			}

			// Perform validation that scroll can be used on item.
			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed to locate equipment being scrolled"))
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed to locate equipment being scrolled"))
			}
			ok = ValidateScrollUse(l)(ctx)(c, scrollItem, *sm.Equipable)
			if !ok {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed slot validation"))
			}

			reserves = append(reserves, inventory.Reserves{
				Slot:     scrollSlot,
				ItemId:   scrollItem.ItemId(),
				Quantity: 1,
			})

			l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
			t, _ := topic.EnvProvider(l)(inventory3.EnvEventInventoryChanged)()
			validator := once.ReservationValidator(transactionId, scrollItem.ItemId())
			handler := inventory.Consume(ConsumeScroll(transactionId, characterId, &scrollItem, equipSlot, whiteScrollItem, legendarySpirit))
			_, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

			err = inventory.RequestReserve(l)(ctx)(transactionId, characterId, inventory2.TypeValueUse, reserves)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, err)
			}
			return nil
		}
	}
}

func ValidateScrollUse(l logrus.FieldLogger) func(ctx context.Context) func(c character.Model, scrollItem item.Model, equipItem equipable2.Model) bool {
	return func(ctx context.Context) func(c character.Model, scrollItem item.Model, equipItem equipable2.Model) bool {
		return func(c character.Model, scrollItem item.Model, equipItem equipable2.Model) bool {
			if item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) {
				// If the scroll is a clean slate scroll, make sure we're not attempting to add mores lots than originally available.
				es, err := statistic.GetById(l)(ctx)(equipItem.ItemId())
				if err != nil {
					return false
				}
				if equipItem.Level() >= byte(es.Slots()) {
					return false
				}
				return true
			} else if IsNotSlotConsumingScroll(item2.Id(scrollItem.ItemId())) {
				// Skip slot validation.
				return true
			} else {
				// If a regular scroll ensure we have an open slot.
				return equipItem.Slots() > 0
			}
		}
	}
}

func IsNotSlotConsumingScroll(id item2.Id) bool {
	return item2.IsScrollSpikes(id) || item2.IsScrollColdProtection(id)
}

func ConsumeScroll(transactionId uuid.UUID, characterId uint32, scrollItem *item.Model, equipSlot int16, whiteScrollItem *item.Model, legendarySpirit bool) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			whiteScroll := whiteScrollItem != nil

			l.Debugf("Character [%d] has reserved items. Consume scroll in slot [%d]. Using white scroll [%t].", characterId, scrollItem.Slot(), whiteScroll)
			c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}

			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), errors.New("equipment not found"))
			}

			ok = ValidateScrollUse(l)(ctx)(c, *scrollItem, *sm.Equipable)
			if !ok {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), errors.New("failed slot validation"))
			}

			ci, err := GetById(l)(ctx)(scrollItem.ItemId())
			if err != nil {
				return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}

			// TODO consume vega scroll
			successProb := ci.success

			// TODO spikes / cursed property
			successRoll := rand.Int31n(100)
			isSuccess := successRoll <= int32(successProb)

			isCursed := false
			passFail := ""
			if isSuccess {
				passFail = "passed"
			} else {
				passFail = "failed"
			}
			l.Debugf("Character [%d] has [%s] scroll [%d]. Rolled [%d]. Needed [%d].", characterId, passFail, scrollItem.ItemId(), successRoll, successProb)
			changes := make([]equipable.Change, 0)
			if isSuccess {
				if item2.IsScrollSpikes(item2.Id(scrollItem.ItemId())) {
					changes = append(changes, equipable.SetSpike())
				} else if item2.IsScrollColdProtection(item2.Id(scrollItem.ItemId())) {
					changes = append(changes, equipable.SetCold())
				} else if item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) {
					changes = append(changes, equipable.AddSlots(1))
				} else if item2.IsChaosScroll(item2.Id(scrollItem.ItemId())) {
					ccs, err := applyChaos(sm.Equipable)
					if err != nil {
						return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
					}
					changes = append(changes, ccs...)
					changes = append(changes,
						equipable.AddSlots(-1),
						equipable.AddLevel(1))
				} else {
					changes = append(changes,
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
						equipable.AddLevel(1))
				}
			} else {
				if !item2.IsScrollSpikes(item2.Id(scrollItem.ItemId())) && !item2.IsScrollColdProtection(item2.Id(scrollItem.ItemId())) && !item2.IsScrollCleanSlate(item2.Id(scrollItem.ItemId())) && !whiteScroll {
					changes = append(changes, equipable.AddSlots(-1))

				}
				if rand.Int31n(100) <= int32(ci.cursed) {
					l.Debugf("Character [%d] item has been cursed.", characterId)
					isCursed = true
				}
			}

			if len(changes) > 0 {
				l.Debugf("Applying [%d] changes to character [%d] item [%d].", len(changes), characterId, sm.Equipable.ItemId())
				err = equipable.ChangeStat(l)(ctx)(characterId, sm.Equipable.ItemId(), sm.Equipable.Slot(), changes...)
				if err != nil {
					return ConsumeError(l)(ctx)(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
				}
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.ItemId(), characterId)
			}
			if whiteScroll {
				err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, whiteScrollItem.Slot())
				if err != nil {
					l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", whiteScrollItem.ItemId(), characterId)
				}
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

func applyChaos(m *equipable2.Model) ([]equipable.Change, error) {
	currents := make([]uint16, 0)
	changers := make([]func(int16) equipable.Change, 0)
	currents = append(currents, m.Strength(), m.Dexterity(), m.Intelligence(), m.Luck(), m.WeaponAttack(), m.WeaponDefense(), m.MagicAttack(), m.MagicDefense(), m.Accuracy(), m.Avoidability(), m.Speed(), m.Jump(), m.HP(), m.MP())
	changers = append(changers, equipable.AddStrength, equipable.AddDexterity, equipable.AddIntelligence, equipable.AddLuck, equipable.AddWeaponAttack, equipable.AddWeaponDefense, equipable.AddMagicAttack, equipable.AddMagicDefense, equipable.AddAccuracy, equipable.AddAvoidability, equipable.AddSpeed, equipable.AddJump, equipable.AddHP, equipable.AddMP)
	return generateChaosChanges(currents, changers)
}

func generateChaosChanges(current []uint16, changers []func(int16) equipable.Change) ([]equipable.Change, error) {
	if len(current) != len(changers) {
		return nil, errors.New("input slices must be of the same length")
	}
	var changes []equipable.Change

	for i, value := range current {
		if value == 0 {
			continue
		}
		adjustment := rollStatAdjustment()
		// TODO maybe structure this better, but for now assume last two items are hp/mp
		if i == len(current)-1 || i == len(current)-2 {
			adjustment *= 10
		}
		change := changers[i](adjustment)
		changes = append(changes, change)
	}

	return changes, nil
}

func rollStatAdjustment() int16 {
	roll := rand.Intn(10000)
	switch {
	case roll < 494: // 4.94%
		return -5
	case roll < 791: // +2.97%
		return -4
	case roll < 1156: // +3.65%
		return -3
	case roll < 1956: // +8.00%
		return -2
	case roll < 3326: // +13.70%
		return -1
	case roll < 5164: // +18.38%
		return 0
	case roll < 7095: // +19.31%
		return 1
	case roll < 8682: // +15.87%
		return 2
	case roll < 9703: // +10.21%
		return 3
	case roll < 9901: // +1.98%
		return 4
	default: // remaining 0.99%
		return 5
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
