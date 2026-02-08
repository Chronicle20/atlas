package consumable

import (
	"atlas-consumables/asset"
	"atlas-consumables/cash"
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/compartment"
	consumable3 "atlas-consumables/data/consumable"
	equipable2 "atlas-consumables/data/equipable"
	_map3 "atlas-consumables/data/map"
	"atlas-consumables/equipable"
	"atlas-consumables/inventory"
	compartment2 "atlas-consumables/kafka/message/compartment"
	"atlas-consumables/kafka/message/consumable"
	once "atlas-consumables/kafka/once/compartment"
	"atlas-consumables/kafka/producer"
	"atlas-consumables/map"
	character2 "atlas-consumables/map/character"
	"atlas-consumables/monster"
	"atlas-consumables/monster/drop/position"
	"atlas-consumables/pet"
	"context"
	"errors"
	"math"
	"math/rand"

	"github.com/Chronicle20/atlas-constants/channel"
	ts "github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	item2 "github.com/Chronicle20/atlas-constants/item"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var ErrPetCannotConsume = errors.New("pet cannot consume")

type ItemConsumer func(l logrus.FieldLogger) func(ctx context.Context) error

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  *character.Processor
	ip  *inventory.Processor
	cpp *compartment.Processor
	cdp *consumable3.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		ip:  inventory.NewProcessor(l, ctx),
		cpp: compartment.NewProcessor(l, ctx),
		cdp: consumable3.NewProcessor(l, ctx),
	}
	return p
}

// ApplyItemEffects applies the effects of a consumable item to a character.
// This is the shared logic used by both regular item consumption and NPC-initiated item use.
// It handles stat buffs (accuracy, evasion, attack, defense, speed, jump) and HP/MP recovery.
func ApplyItemEffects(l logrus.FieldLogger, ctx context.Context, c character.Model, f field.Model, ci consumable3.Model, characterId uint32, itemId item2.Id) {
	bp := buff.NewProcessor(l, ctx)
	cp := character.NewProcessor(l, ctx)

	statups := make([]stat.Model, 0)
	duration := int32(0)

	if val, ok := ci.GetSpec(consumable3.SpecTypeAccuracy); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeAccuracy,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeEvasion); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeAvoidability,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeHP); ok && val > 0 {
		_ = cp.ChangeHP(f, characterId, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeHPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		res := int16(math.Floor(float64(c.MaxHp()) * pct))
		_ = cp.ChangeHP(f, characterId, res)
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeJump); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeJump,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicAttack); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeMagicAttack,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicDefense); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeMagicDefense,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMP); ok && val > 0 {
		_ = cp.ChangeMP(f, characterId, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		res := int16(math.Floor(float64(c.MaxMp()) * pct))
		_ = cp.ChangeMP(f, characterId, res)
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponAttack); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeWeaponAttack,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponDefense); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeWeaponDefense,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeSpeed); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeSpeed,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
		statups = append(statups, stat.Model{
			Type:   ts.TemporaryStatTypeMorph,
			Amount: val,
		})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeTime); ok && val > 0 {
		duration = val / 1000
	}

	if len(statups) > 0 {
		_ = bp.Apply(f, characterId, -int32(itemId), duration, statups)(characterId)
	}
}

func (p *Processor) RequestItemConsume(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16) error {
	transactionId := uuid.New()
	p.l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, uint32(itemId))

	it, ok := inventory2.TypeFromItemId(itemId)
	if !ok {
		return errors.New("invalid item id")
	}

	var itemConsumer ItemConsumer
	if item2.GetClassification(itemId) == item2.Classification(200) || item2.GetClassification(itemId) == item2.Classification(201) || item2.GetClassification(itemId) == item2.Classification(202) {
		itemConsumer = ConsumeStandard(transactionId, characterId, slot, itemId)
	} else if item2.GetClassification(itemId) == item2.ClassificationConsumableTownWarp {
		itemConsumer = ConsumeTownScroll(transactionId, characterId, slot, itemId)
	} else if item2.GetClassification(itemId) == item2.ClassificationConsumablePetFood {
		itemConsumer = ConsumePetFood(transactionId, characterId, slot, itemId)
	} else if item2.GetClassification(itemId) == item2.ClassificationPetConsumable {
		itemConsumer = ConsumeCashPetFood(transactionId, characterId, slot, itemId)
	} else if item2.GetClassification(itemId) == item2.ClassificationConsumableSummoningSack {
		itemConsumer = ConsumeSummoningSack(transactionId, c, characterId, slot, itemId)
	}

	handler := compartment.Consume(itemConsumer)

	_, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

	err = p.cpp.RequestReserve(transactionId, characterId, it, []compartment.Reserves{{Slot: slot, ItemId: uint32(itemId), Quantity: quantity}})
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) ConsumeError(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error {
	p.l.Debugf("Character [%d] unable to consume item due to error: [%v]", characterId, err)
	cErr := p.cpp.CancelItemReservation(characterId, inventoryType, transactionId, slot)
	if cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to cancel item reservation at inventory [%d] slot [%d] for character [%d] as part of transaction [%d].", inventoryType, slot, characterId, transactionId)
	}

	errorType := ""
	if errors.Is(err, ErrPetCannotConsume) {
		errorType = consumable.ErrorTypePetCannotConsume
	}

	cErr = producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), errorType))
	if cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to issue consumption error [%v] on event topic. Character [%d] likely going to be stuck.", err, characterId)
	}
	return err
}

func ConsumeStandard(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)

			c, err := cp.GetById()(characterId)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			m, err := character2.NewProcessor(l, ctx).GetMap(characterId)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := p.cdp.GetById(uint32(itemId))
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			err = compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ApplyItemEffects(l, ctx, c, m, ci, characterId, itemId)
			return nil
		}
	}
}

func ConsumeTownScroll(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			m, err := character2.NewProcessor(l, ctx).GetMap(characterId)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := p.cdp.GetById(uint32(itemId))
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			toMapId := _map2.EmptyMapId
			if val, ok := ci.GetSpec(consumable3.SpecTypeMoveTo); ok && val > 0 {
				toMapId = _map2.Id(val)
			}
			if toMapId == _map2.EmptyMapId {
				mm, err := _map3.NewProcessor(l, ctx).GetById(m.MapId())
				if err != nil {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
				}
				toMapId = mm.ReturnMapId()
			}

			err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			toField := field.NewBuilder(m.WorldId(), m.ChannelId(), toMapId).SetInstance(m.Instance()).Build()
			err = _map.NewProcessor(l, ctx).WarpRandom(toField)(characterId)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func ConsumePetFood(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			pp := pet.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			pe, err := pp.HungriestByOwnerProvider(characterId)()
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := p.cdp.GetById(uint32(itemId))
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			inc := byte(0)
			if val, ok := ci.GetSpec(consumable3.SpecTypeInc); ok {
				inc = byte(val)
			}

			err = pp.AwardFullness(characterId, pe.Id(), inc)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

func ConsumeCashPetFood(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			pp := pet.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			ci, err := cash.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
			}

			hpp := pp.HungryByOwnerProvider(characterId)
			fhpp := model.FilteredProvider(hpp, model.Filters[pet.Model](pet.IsTemplateFilter(ci.Indexes()...)))
			pe, err := pet.HungriestToOneProvider(fhpp)()
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, ErrPetCannotConsume)
			}

			inc := byte(0)
			if val, ok := ci.GetSpec(cash.SpecTypeInc); ok {
				inc = byte(val)
			}

			err = pp.AwardFullness(characterId, pe.Id(), inc)
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

func ConsumeSummoningSack(transactionId uuid.UUID, ch channel.Model, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			c, err := character.NewProcessor(l, ctx).GetById()(characterId)
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ci, err := consumable3.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			pos, err := position.NewProcessor(l, ctx).GetInMap(c.MapId(), c.X(), c.Y(), c.X(), c.Y())()
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			l.Debugf("Character [%d] summoning [%d] monsters at [%d,%d]. They are at [%d,%d].", characterId, len(ci.MonsterSummons()), pos.X(), pos.Y(), c.X(), c.Y())
			for _, msm := range ci.MonsterSummons() {
				roll := uint32(rand.Int31n(100))
				if roll < msm.Probability() {
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
					err = monster.NewProcessor(l, ctx).CreateMonster(f, msm.TemplateId(), pos.X(), pos.Y(), 0, 0)
					if err != nil {
						l.WithError(err).Errorf("Unable to summon monster [%d] for character [%d] summoning bag.", msm.TemplateId(), characterId)
					} else {
						l.Debugf("Character [%d] use of summoning sack [%d] spawned monster [%d] at [%d,%d].", characterId, itemId, msm.TemplateId(), c.X(), c.Y())
					}
				}
			}

			err = compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

func (p *Processor) RequestScroll(characterId uint32, scrollSlot int16, equipSlot int16, whiteScroll bool, legendarySpirit bool) error {
	cp := character.NewProcessor(p.l, p.ctx)
	cpp := compartment.NewProcessor(p.l, p.ctx)

	transactionId := uuid.New()
	reserves := make([]compartment.Reserves, 0)

	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, err)
	}

	scrollItem, ok := c.Inventory().Consumable().FindBySlot(scrollSlot)
	if !ok {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("scroll item not found"))
	}

	// Ensure if we are using a "white scroll" that we have a white scroll item.
	var whiteScrollItem *asset.Model
	if whiteScroll {
		whiteScrollItem, ok = c.Inventory().Consumable().FindFirstByItemId(2340000)
		if !ok {
			return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("does not have white scroll"))
		}
		reserves = append(reserves, compartment.Reserves{
			Slot:     whiteScrollItem.Slot(),
			ItemId:   whiteScrollItem.TemplateId(),
			Quantity: 1,
		})
	}

	// Perform validation that scroll can be used on item.
	s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
	if err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed to locate equipment being scrolled"))
	}
	sm, ok := c.Equipment().Get(s.Type)
	if !ok || sm.Equipable == nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed to locate equipment being scrolled"))
	}
	ok = p.ValidateScrollUse(*scrollItem, *sm.Equipable)
	if !ok {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, errors.New("failed slot validation"))
	}

	reserves = append(reserves, compartment.Reserves{
		Slot:     scrollSlot,
		ItemId:   scrollItem.TemplateId(),
		Quantity: 1,
	})

	p.l.Debugf("Creating OneTime topic consumer to await transaction [%s] completion or cancellation.", transactionId.String())
	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, scrollItem.TemplateId())
	handler := compartment.Consume(ConsumeScroll(transactionId, characterId, scrollItem, equipSlot, whiteScrollItem, legendarySpirit))
	_, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

	err = cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, reserves)
	if err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollSlot, err)
	}
	return nil
}

func (p *Processor) ValidateScrollUse(scrollItem asset.Model, equipItem asset.Model) bool {
	ep := equipable2.NewProcessor(p.l, p.ctx)
	if item2.IsScrollCleanSlate(item2.Id(scrollItem.TemplateId())) {
		// If the scroll is a clean slate scroll, make sure we're not attempting to add mores lots than originally available.
		es, err := ep.GetById(equipItem.TemplateId())
		if err != nil {
			return false
		}
		if equipItem.Level() >= byte(es.Slots()) {
			return false
		}
		return true
	} else if IsNotSlotConsumingScroll(item2.Id(scrollItem.TemplateId())) {
		// Skip slot validation.
		return true
	} else {
		// If a regular scroll ensure we have an open slot.
		return equipItem.Slots() > 0
	}
}

func IsNotSlotConsumingScroll(id item2.Id) bool {
	return item2.IsScrollSpikes(id) || item2.IsScrollColdProtection(id)
}

func ConsumeScroll(transactionId uuid.UUID, characterId uint32, scrollItem *asset.Model, equipSlot int16, whiteScrollItem *asset.Model, legendarySpirit bool) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)
			ep := equipable.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			whiteScroll := whiteScrollItem != nil

			l.Debugf("Character [%d] has reserved items. Consume scroll in slot [%d]. Using white scroll [%t].", characterId, scrollItem.Slot(), whiteScroll)
			c, err := cp.GetById(cp.InventoryDecorator)(characterId)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}

			s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}
			sm, ok := c.Equipment().Get(s.Type)
			if !ok || sm.Equipable == nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), errors.New("equipment not found"))
			}

			ok = p.ValidateScrollUse(*scrollItem, *sm.Equipable)
			if !ok {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), errors.New("failed slot validation"))
			}

			ci, err := p.cdp.GetById(scrollItem.TemplateId())
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}

			// TODO consume vega scroll
			successProb := ci.SuccessRate()

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
			l.Debugf("Character [%d] has [%s] scroll [%d]. Rolled [%d]. Needed [%d].", characterId, passFail, scrollItem.TemplateId(), successRoll, successProb)
			changes := make([]equipable.Change, 0)
			if isSuccess {
				if item2.IsScrollSpikes(item2.Id(scrollItem.TemplateId())) {
					changes = append(changes, equipable.SetSpike())
				} else if item2.IsScrollColdProtection(item2.Id(scrollItem.TemplateId())) {
					changes = append(changes, equipable.SetCold())
				} else if item2.IsScrollCleanSlate(item2.Id(scrollItem.TemplateId())) {
					changes = append(changes, equipable.AddSlots(1))
				} else if item2.IsChaosScroll(item2.Id(scrollItem.TemplateId())) {
					ccs, err := applyChaos(*sm.Equipable)
					if err != nil {
						return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
					}
					changes = append(changes, ccs...)
					changes = append(changes,
						equipable.AddSlots(-1),
						equipable.AddLevel(1))
				} else {
					changes = append(changes,
						equipable.AddStrength(int16(ci.StrengthIncrease())),
						equipable.AddDexterity(int16(ci.DexterityIncrease())),
						equipable.AddIntelligence(int16(ci.IntelligenceIncrease())),
						equipable.AddLuck(int16(ci.LuckIncrease())),
						equipable.AddHp(int16(ci.MaxHPIncrease())),
						equipable.AddMp(int16(ci.MaxMPIncrease())),
						equipable.AddWeaponAttack(int16(ci.WeaponAttackIncrease())),
						equipable.AddMagicAttack(int16(ci.MagicAttackIncrease())),
						equipable.AddWeaponDefense(int16(ci.WeaponDefenseIncrease())),
						equipable.AddMagicDefense(int16(ci.MagicDefenseIncrease())),
						equipable.AddAccuracy(int16(ci.AccuracyIncrease())),
						equipable.AddAvoidability(int16(ci.AvoidabilityIncrease())),
						equipable.AddHands(int16(ci.HandsIncrease())),
						equipable.AddSpeed(int16(ci.SpeedIncrease())),
						equipable.AddJump(int16(ci.JumpIncrease())),
						equipable.AddSlots(-1),
						equipable.AddLevel(1))
				}
			} else {
				if !item2.IsScrollSpikes(item2.Id(scrollItem.TemplateId())) && !item2.IsScrollColdProtection(item2.Id(scrollItem.TemplateId())) && !item2.IsScrollCleanSlate(item2.Id(scrollItem.TemplateId())) && !whiteScroll {
					changes = append(changes, equipable.AddSlots(-1))

				}
				if rand.Int31n(100) <= int32(ci.CursedRate()) {
					l.Debugf("Character [%d] item has been cursed.", characterId)
					isCursed = true
				}
			}

			if len(changes) > 0 {
				l.Debugf("Applying [%d] changes to character [%d] item [%d].", len(changes), characterId, sm.Equipable.TemplateId())

				err = ep.ChangeStat(characterId, transactionId, *sm.Equipable, changes...)
				if err != nil {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
				}
			}

			err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.TemplateId(), characterId)
			}
			if whiteScroll {
				err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, whiteScrollItem.Slot())
				if err != nil {
					l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", whiteScrollItem.TemplateId(), characterId)
				}
			}

			if isCursed {
				err = cpp.DestroyItem(characterId, inventory2.TypeValueEquip, equipSlot)
				if err != nil {
					l.WithError(err).Errorf("Unable to destroy item in slot [%d] for character [%d] during scrolling.", equipSlot, characterId)
				}
			}

			if isSuccess {
				_ = p.PassScroll(characterId, legendarySpirit, whiteScroll)
			} else {
				_ = p.FailScroll(characterId, isCursed, legendarySpirit, whiteScroll)
			}
			return nil
		}
	}
}

type chaosStat struct {
	value      uint16
	changer    func(int16) equipable.Change
	multiplier int16
}

func applyChaos(m asset.Model) ([]equipable.Change, error) {
	stats := []chaosStat{
		{m.Strength(), equipable.AddStrength, 1},
		{m.Dexterity(), equipable.AddDexterity, 1},
		{m.Intelligence(), equipable.AddIntelligence, 1},
		{m.Luck(), equipable.AddLuck, 1},
		{m.WeaponAttack(), equipable.AddWeaponAttack, 1},
		{m.WeaponDefense(), equipable.AddWeaponDefense, 1},
		{m.MagicAttack(), equipable.AddMagicAttack, 1},
		{m.MagicDefense(), equipable.AddMagicDefense, 1},
		{m.Accuracy(), equipable.AddAccuracy, 1},
		{m.Avoidability(), equipable.AddAvoidability, 1},
		{m.Speed(), equipable.AddSpeed, 1},
		{m.Jump(), equipable.AddJump, 1},
		{m.Hp(), equipable.AddHp, 10},
		{m.Mp(), equipable.AddMp, 10},
	}
	return generateChaosChanges(stats)
}

func generateChaosChanges(stats []chaosStat) ([]equipable.Change, error) {
	var changes []equipable.Change
	for _, s := range stats {
		if s.value == 0 {
			continue
		}
		change := s.changer(rollStatAdjustment() * s.multiplier)
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

func (p *Processor) PassScroll(characterId uint32, legendarySpirit bool, whiteScroll bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ScrollEventProvider(ts.Id(characterId))(true, false, legendarySpirit, whiteScroll))
}

// ApplyConsumableEffect applies item effects to a character without consuming from inventory
// This is used for NPC-initiated buffs (e.g., NPC blessings via cm.useItem())
func (p *Processor) ApplyConsumableEffect(transactionId uuid.UUID, _ channel.Model, characterId uint32, itemId item2.Id) error {
	cp := character.NewProcessor(p.l, p.ctx)

	c, err := cp.GetById()(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get character [%d] for apply consumable effect", characterId)
		return err
	}

	m, err := character2.NewProcessor(p.l, p.ctx).GetMap(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get map for character [%d] for apply consumable effect", characterId)
		return err
	}

	ci, err := p.cdp.GetById(uint32(itemId))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get consumable data for item [%d]", itemId)
		return err
	}

	ApplyItemEffects(p.l, p.ctx, c, m, ci, characterId, itemId)
	p.l.Debugf("Successfully applied consumable [%d] effects to character [%d]", itemId, characterId)

	// Emit the effect applied event for saga completion
	err = producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(EffectAppliedEventProvider(ts.Id(characterId), itemId, transactionId))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to emit effect applied event for character [%d] item [%d]", characterId, itemId)
		return err
	}
	return nil
}

// CancelConsumableEffect cancels the buff effects of a consumable item on a character.
// This sends a cancel command to the buff service using sourceId = -int32(itemId).
func (p *Processor) CancelConsumableEffect(_ uuid.UUID, characterId uint32, itemId item2.Id, f field.Model) error {
	bp := buff.NewProcessor(p.l, p.ctx)
	sourceId := -int32(itemId)
	err := bp.Cancel(f, characterId, sourceId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to cancel consumable effect [%d] for character [%d]", itemId, characterId)
		return err
	}
	p.l.Debugf("Successfully cancelled consumable [%d] effects for character [%d]", itemId, characterId)
	return nil
}

func (p *Processor) FailScroll(characterId uint32, cursed bool, legendarySpirit bool, whiteScroll bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ScrollEventProvider(ts.Id(characterId))(false, cursed, legendarySpirit, whiteScroll))
}
