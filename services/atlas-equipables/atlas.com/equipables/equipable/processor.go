package equipable

import (
	"atlas-equipables/data/equipable"
	"atlas-equipables/database"
	"atlas-equipables/kafka/message"
	equipable2 "atlas-equipables/kafka/message/equipable"
	"atlas-equipables/kafka/producer"
	"context"
	"math"
	"math/rand"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                   logrus.FieldLogger
	ctx                 context.Context
	db                  *gorm.DB
	t                   tenant.Model
	edp                 *equipable.Processor
	GetById             func(id uint32) (Model, error)
	CreateAndEmit       func(i Model) (Model, error)
	CreateRandomAndEmit func(id uint32) (Model, error)
	UpdateAndEmit       func(i Model) (Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		edp: equipable.NewProcessor(l, ctx),
	}
	p.GetById = model.CollapseProvider(p.ByIdModelProvider)
	p.CreateAndEmit = message.EmitWithResult[Model, Model](producer.ProviderImpl(l)(ctx))(p.Create)
	p.CreateRandomAndEmit = message.EmitWithResult[Model, uint32](producer.ProviderImpl(l)(ctx))(p.CreateRandom)
	p.UpdateAndEmit = message.EmitWithResult[Model, Model](producer.ProviderImpl(l)(ctx))(p.Update)
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:                   p.l,
		ctx:                 p.ctx,
		db:                  db,
		t:                   p.t,
		edp:                 p.edp,
		GetById:             p.GetById,
		CreateAndEmit:       p.CreateAndEmit,
		CreateRandomAndEmit: p.CreateRandomAndEmit,
		UpdateAndEmit:       p.UpdateAndEmit,
	}
}

func (p *Processor) ByIdModelProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(byIdEntityProvider(p.t.Id(), id)(p.db))
}

func (p *Processor) Create(mb *message.Buffer) func(i Model) (Model, error) {
	return func(i Model) (Model, error) {
		p.l.Debugf("Creating equipable for item [%d].", i.ItemId())
		var o Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var err error
			if i.Strength() == 0 && i.Dexterity() == 0 && i.Intelligence() == 0 && i.Luck() == 0 && i.HP() == 0 && i.MP() == 0 && i.WeaponAttack() == 0 && i.WeaponDefense() == 0 && i.MagicAttack() == 0 && i.MagicDefense() == 0 && i.Accuracy() == 0 && i.Avoidability() == 0 && i.Hands() == 0 && i.Speed() == 0 && i.Jump() == 0 && i.Slots() == 0 {
				var ea equipable.Model
				ea, err = p.edp.GetById(i.ItemId())
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get equipable information for %d.", i.ItemId())
					return err
				}

				o, err = create(p.db, p.t.Id(), i.ItemId(), ea.Strength(), ea.Dexterity(), ea.Intelligence(), ea.Luck(), ea.HP(), ea.MP(), ea.WeaponAttack(), ea.MagicAttack(), ea.WeaponDefense(), ea.MagicDefense(), ea.Accuracy(), ea.Avoidability(), ea.Hands(), ea.Speed(), ea.Jump(), ea.Slots())
				if err != nil {
					return err
				}
				return mb.Put(equipable2.EnvStatusEventTopic, CreatedStatusEventProvider(o))
			}
			o, err = create(p.db, p.t.Id(), i.ItemId(), i.Strength(), i.Dexterity(), i.Intelligence(), i.Luck(), i.HP(), i.MP(), i.WeaponAttack(), i.MagicAttack(), i.WeaponDefense(), i.MagicDefense(), i.Accuracy(), i.Avoidability(), i.Hands(), i.Speed(), i.Jump(), i.Slots())
			if err != nil {
				return err
			}
			return mb.Put(equipable2.EnvStatusEventTopic, CreatedStatusEventProvider(o))
		})
		if txErr != nil {
			return o, txErr
		}
		p.l.Debugf("Equipable [%d] created from item [%d] template.", o.Id(), o.ItemId())
		return o, nil
	}
}

func (p *Processor) CreateRandom(mb *message.Buffer) func(itemId uint32) (Model, error) {
	return func(itemId uint32) (Model, error) {
		p.l.Debugf("Creating equipable for item [%d].", itemId)
		ea, err := p.edp.GetById(itemId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get equipable information for %d.", itemId)
			return Model{}, err
		}
		strength := getRandomStat(ea.Strength(), 5)
		dexterity := getRandomStat(ea.Dexterity(), 5)
		intelligence := getRandomStat(ea.Intelligence(), 5)
		luck := getRandomStat(ea.Luck(), 5)
		hp := getRandomStat(ea.HP(), 10)
		mp := getRandomStat(ea.MP(), 10)
		weaponAttack := getRandomStat(ea.WeaponAttack(), 5)
		magicAttack := getRandomStat(ea.MagicAttack(), 5)
		weaponDefense := getRandomStat(ea.WeaponDefense(), 10)
		magicDefense := getRandomStat(ea.MagicDefense(), 10)
		accuracy := getRandomStat(ea.Accuracy(), 5)
		avoidability := getRandomStat(ea.Avoidability(), 5)
		hands := getRandomStat(ea.Hands(), 5)
		speed := getRandomStat(ea.Speed(), 5)
		jump := getRandomStat(ea.Jump(), 5)
		slots := ea.Slots()
		o, err := create(p.db, p.t.Id(), itemId, strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots)
		if err != nil {
			return Model{}, err
		}
		err = mb.Put(equipable2.EnvStatusEventTopic, CreatedStatusEventProvider(o))
		if err != nil {
			return Model{}, err
		}
		return o, nil
	}
}

func getRandomStat(defaultValue uint16, max uint16) uint16 {
	if defaultValue == 0 {
		return 0
	}
	maxRange := math.Min(math.Ceil(float64(defaultValue)*0.1), float64(max))
	return uint16(float64(defaultValue)-maxRange) + uint16(math.Floor(rand.Float64()*(maxRange*2.0+1.0)))
}

func (p *Processor) Update(mb *message.Buffer) func(i Model) (Model, error) {
	return func(i Model) (Model, error) {
		var um Model
		p.l.Debugf("Updating equipable [%d].", i.Id())
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById(i.Id())
			if err != nil {
				return err
			}
			updates := make(map[string]interface{})
			if i.Strength() != c.Strength() {
				updates["strength"] = i.Strength()
			}
			if i.Dexterity() != c.Dexterity() {
				updates["dexterity"] = i.Dexterity()
			}
			if i.Intelligence() != c.Intelligence() {
				updates["intelligence"] = i.Intelligence()
			}
			if i.Luck() != c.Luck() {
				updates["luck"] = i.Luck()
			}
			if i.HP() != c.HP() {
				updates["hp"] = i.HP()
			}
			if i.MP() != c.MP() {
				updates["mp"] = i.MP()
			}
			if i.WeaponAttack() != c.WeaponAttack() {
				updates["weapon_attack"] = i.WeaponAttack()
			}
			if i.MagicAttack() != c.MagicAttack() {
				updates["magic_attack"] = i.MagicAttack()
			}
			if i.WeaponDefense() != c.WeaponDefense() {
				updates["weapon_defense"] = i.WeaponDefense()
			}
			if i.MagicDefense() != c.MagicDefense() {
				updates["magic_defense"] = i.MagicDefense()
			}
			if i.Accuracy() != c.Accuracy() {
				updates["accuracy"] = i.Accuracy()
			}
			if i.Avoidability() != c.Avoidability() {
				updates["avoidability"] = i.Avoidability()
			}
			if i.Hands() != c.Hands() {
				updates["hands"] = i.Hands()
			}
			if i.Speed() != c.Speed() {
				updates["speed"] = i.Speed()
			}
			if i.Jump() != c.Jump() {
				updates["jump"] = i.Jump()
			}
			if i.Slots() != c.Slots() {
				updates["slots"] = i.Slots()
			}
			if i.OwnerName() != c.OwnerName() {
				updates["owner_name"] = i.OwnerName()
			}
			if i.Locked() != c.Locked() {
				updates["locked"] = i.Locked()
			}
			if i.Spikes() != c.Spikes() {
				updates["spikes"] = i.Spikes()
			}
			if i.KarmaUsed() != c.KarmaUsed() {
				updates["karma_used"] = i.KarmaUsed()
			}
			if i.Cold() != c.Cold() {
				updates["cold"] = i.Cold()
			}
			if i.CanBeTraded() != c.CanBeTraded() {
				updates["can_be_traded"] = i.CanBeTraded()
			}
			if i.LevelType() != c.LevelType() {
				updates["level_type"] = i.LevelType()
			}
			if i.Level() != c.Level() {
				updates["level"] = i.Level()
			}
			if i.Experience() != c.Experience() {
				updates["experience"] = i.Experience()
			}
			if i.HammersApplied() != c.HammersApplied() {
				updates["hammers_applied"] = i.HammersApplied()
			}
			if i.Expiration() != c.Expiration() {
				updates["expiration"] = i.Expiration()
			}
			um, err = update(tx, p.t.Id(), i.Id(), updates)
			if err != nil {
				return err
			}
			return mb.Put(equipable2.EnvStatusEventTopic, UpdatedStatusEventProvider(um))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return um, nil
	}
}

func (p *Processor) DeleteByIdAndEmit(id uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(model.Flip(p.DeleteById)(id))
}

// MarkEquipped sets the equippedSince timestamp to now for the given equipment
func (p *Processor) MarkEquipped(id uint32) error {
	p.l.Debugf("Marking equipment [%d] as equipped.", id)
	return setEquipped(p.db, p.t.Id(), id)
}

// MarkUnequipped clears the equippedSince timestamp for the given equipment
func (p *Processor) MarkUnequipped(id uint32) error {
	p.l.Debugf("Marking equipment [%d] as unequipped.", id)
	return clearEquipped(p.db, p.t.Id(), id)
}

func (p *Processor) DeleteById(mb *message.Buffer) func(id uint32) error {
	return func(id uint32) error {
		p.l.Debugf("Attempting to delete equipable [%d].", id)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			err := deleteById(p.db, p.t.Id(), id)
			if err != nil {
				return err
			}
			return mb.Put(equipable2.EnvStatusEventTopic, DeletedStatusEventProvider(id))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete equipable [%d].", id)
			return txErr
		}
		p.l.Debugf("Equipable [%d] deleted.", id)
		return nil
	}
}
