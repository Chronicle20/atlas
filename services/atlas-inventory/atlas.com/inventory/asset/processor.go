package asset

import (
	"atlas-inventory/data/consumable"
	"atlas-inventory/data/equipment/statistics"
	"atlas-inventory/data/etc"
	"atlas-inventory/data/setup"
	"atlas-inventory/database"
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/asset"
	"atlas-inventory/kafka/producer"
	"atlas-inventory/pet"
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                   logrus.FieldLogger
	ctx                 context.Context
	db                  *gorm.DB
	t                   tenant.Model
	petProcessor        *pet.Processor
	consumableProcessor consumable.Processor
	setupProcessor      *setup.Processor
	etcProcessor        *etc.Processor
	statProcessor       *statistics.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:                   l,
		ctx:                 ctx,
		db:                  db,
		t:                   tenant.MustFromContext(ctx),
		petProcessor:        pet.NewProcessor(l, ctx),
		consumableProcessor: consumable.NewProcessor(l, ctx),
		setupProcessor:      setup.NewProcessor(l, ctx),
		etcProcessor:        etc.NewProcessor(l, ctx),
		statProcessor:       statistics.NewProcessor(l, ctx),
	}
}

func (p *Processor) WithTransaction(tx *gorm.DB) *Processor {
	return &Processor{
		l:                   p.l,
		ctx:                 p.ctx,
		db:                  tx,
		t:                   p.t,
		petProcessor:        p.petProcessor,
		consumableProcessor: p.consumableProcessor,
		setupProcessor:      p.setupProcessor,
		etcProcessor:        p.etcProcessor,
		statProcessor:       p.statProcessor,
	}
}

func (p *Processor) WithConsumableProcessor(conp consumable.Processor) *Processor {
	return &Processor{
		l:                   p.l,
		ctx:                 p.ctx,
		db:                  p.db,
		t:                   p.t,
		petProcessor:        p.petProcessor,
		consumableProcessor: conp,
		setupProcessor:      p.setupProcessor,
		etcProcessor:        p.etcProcessor,
		statProcessor:       p.statProcessor,
	}
}

func (p *Processor) ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByCompartmentId(p.t.Id(), compartmentId)(p.db))(model.ParallelMap())
}

func (p *Processor) GetByCompartmentId(compartmentId uuid.UUID) ([]Model, error) {
	return p.ByCompartmentIdProvider(compartmentId)()
}

func (p *Processor) GetBySlot(compartmentId uuid.UUID, slot int16) (Model, error) {
	return p.BySlotProvider(compartmentId)(slot)()
}

func (p *Processor) BySlotProvider(compartmentId uuid.UUID) func(slot int16) model.Provider[Model] {
	return func(slot int16) model.Provider[Model] {
		return model.Map(Make)(getBySlot(p.t.Id(), compartmentId, slot)(p.db))
	}
}

func (p *Processor) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), id)(p.db))
}

func (p *Processor) GetById(id uint32) (Model, error) {
	return model.CollapseProvider(p.ByIdProvider)(id)
}

// GetSlotMax retrieves the maximum slot capacity for a given asset template.
func (p *Processor) GetSlotMax(templateId uint32) (uint32, error) {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return 0, errors.New("unknown item type")
	}

	switch inventoryType {
	case inventory.TypeValueUse:
		m, err := p.consumableProcessor.GetById(templateId)
		if err != nil {
			return 0, err
		}
		return m.SlotMax(), nil
	case inventory.TypeValueSetup:
		m, err := p.setupProcessor.GetById(templateId)
		if err != nil {
			return 0, err
		}
		return m.SlotMax(), nil
	case inventory.TypeValueETC:
		m, err := p.etcProcessor.GetById(templateId)
		if err != nil {
			return 0, err
		}
		return m.SlotMax(), nil
	default:
		return 1, nil
	}
}

func (p *Processor) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
		return func(a Model) error {
			p.l.Debugf("Attempting to delete asset [%d].", a.Id())
			err := deleteById(p.db, p.t.Id(), a.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete asset [%d].", a.Id())
				return err
			}
			p.l.Debugf("Deleted asset [%d].", a.Id())
			return mb.Put(asset.EnvEventTopicStatus, DeletedEventStatusProvider(transactionId, characterId, compartmentId, a.Id(), a.TemplateId(), a.Slot()))
		}
	}
}

func (p *Processor) Expire(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, isCash bool, replaceItemId uint32, replaceMessage string) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, isCash bool, replaceItemId uint32, replaceMessage string) func(a Model) error {
		return func(a Model) error {
			p.l.Debugf("Attempting to expire asset [%d].", a.Id())
			err := deleteById(p.db, p.t.Id(), a.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to expire asset [%d].", a.Id())
				return err
			}
			p.l.Debugf("Expired asset [%d].", a.Id())
			return mb.Put(asset.EnvEventTopicStatus, ExpiredEventStatusProvider(transactionId, characterId, compartmentId, a.Id(), a.TemplateId(), a.Slot(), isCash, replaceItemId, replaceMessage))
		}
	}
}

func (p *Processor) Drop(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
		return func(a Model) error {
			p.l.Debugf("Attempting to drop asset [%d].", a.Id())
			err := deleteById(p.db, p.t.Id(), a.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to drop asset [%d].", a.Id())
				return err
			}
			p.l.Debugf("Dropped asset [%d].", a.Id())
			return mb.Put(asset.EnvEventTopicStatus, DeletedEventStatusProvider(transactionId, characterId, compartmentId, a.Id(), a.TemplateId(), a.Slot()))
		}
	}
}

func (p *Processor) UpdateSlot(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, ap model.Provider[Model], sp model.Provider[int16]) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, ap model.Provider[Model], sp model.Provider[int16]) error {
		a, err := ap()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err != nil {
			return nil
		}
		s, err := sp()
		if err != nil {
			return err
		}
		p.l.Debugf("Character [%d] attempting to update slot of asset [%d] to [%d] from [%d].", characterId, a.Id(), s, a.Slot())
		err = updateSlot(p.db, p.t.Id(), a.Id(), s)
		if err != nil {
			return err
		}
		if a.Slot() != int16(math.MinInt16) && s != int16(math.MinInt16) {
			return mb.Put(asset.EnvEventTopicStatus, MovedEventStatusProvider(transactionId, characterId, compartmentId, a.Id(), a.TemplateId(), a.Slot(), s, a.CreatedAt()))
		}
		return nil
	}
}

func (p *Processor) UpdateQuantity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, a Model, quantity uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, a Model, quantity uint32) error {
		if !a.HasQuantity() {
			return errors.New("cannot update quantity of non-stackable")
		}
		err := updateQuantity(p.db, p.t.Id(), a.Id(), quantity)
		if err != nil {
			return err
		}
		return mb.Put(asset.EnvEventTopicStatus, QuantityChangedEventStatusProvider(transactionId, characterId, compartmentId, a.Id(), a.TemplateId(), a.Slot(), quantity))
	}
}

func (p *Processor) UpdateEquipmentStats(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats Model) error {
	return func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats Model) error {
		a, err := p.GetById(assetId)
		if err != nil {
			return err
		}

		// Build the updated model with new stats but preserving identity fields
		updated := Clone(a).
			SetStrength(stats.Strength()).
			SetDexterity(stats.Dexterity()).
			SetIntelligence(stats.Intelligence()).
			SetLuck(stats.Luck()).
			SetHp(stats.Hp()).
			SetMp(stats.Mp()).
			SetWeaponAttack(stats.WeaponAttack()).
			SetMagicAttack(stats.MagicAttack()).
			SetWeaponDefense(stats.WeaponDefense()).
			SetMagicDefense(stats.MagicDefense()).
			SetAccuracy(stats.Accuracy()).
			SetAvoidability(stats.Avoidability()).
			SetHands(stats.Hands()).
			SetSpeed(stats.Speed()).
			SetJump(stats.Jump()).
			SetSlots(stats.Slots()).
			SetLocked(stats.Locked()).
			SetSpikes(stats.Spikes()).
			SetKarmaUsed(stats.KarmaUsed()).
			SetCold(stats.Cold()).
			SetCanBeTraded(stats.CanBeTraded()).
			SetLevelType(stats.LevelType()).
			SetLevel(stats.Level()).
			SetExperience(stats.Experience()).
			SetHammersApplied(stats.HammersApplied()).
			SetExpiration(stats.Expiration()).
			Build()

		err = updateEquipmentStats(p.db, p.t.Id(), assetId, updated)
		if err != nil {
			return err
		}
		return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
	}
}

func (p *Processor) DeleteAndEmit(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32) error {
	p.l.Debugf("Attempting to delete and emit asset [%d] for character [%d] in compartment [%s].", assetId, characterId, compartmentId.String())
	a, err := p.GetById(assetId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find asset [%d] for deletion.", assetId)
		return err
	}
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Delete(buf)(transactionId, characterId, compartmentId)(a)
	})
}

func (p *Processor) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64) (Model, error) {
		p.l.Debugf("Character [%d] attempting to create [%d] item(s) [%d] in slot [%d] of compartment [%s].", characterId, quantity, templateId, slot, compartmentId.String())
		var a Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
			if !ok {
				return errors.New("unknown item type")
			}

			b := NewBuilder(compartmentId, templateId).
				SetSlot(slot).
				SetExpiration(expiration).
				SetCreatedAt(time.Now())

			switch inventoryType {
			case inventory.TypeValueEquip:
				ea, err := p.statProcessor.GetById(templateId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get equipment stats for item [%d].", templateId)
					return err
				}
				b.SetStrength(getRandomStat(ea.Strength(), 5)).
					SetDexterity(getRandomStat(ea.Dexterity(), 5)).
					SetIntelligence(getRandomStat(ea.Intelligence(), 5)).
					SetLuck(getRandomStat(ea.Luck(), 5)).
					SetHp(getRandomStat(ea.Hp(), 10)).
					SetMp(getRandomStat(ea.Mp(), 10)).
					SetWeaponAttack(getRandomStat(ea.WeaponAttack(), 5)).
					SetMagicAttack(getRandomStat(ea.MagicAttack(), 5)).
					SetWeaponDefense(getRandomStat(ea.WeaponDefense(), 10)).
					SetMagicDefense(getRandomStat(ea.MagicDefense(), 10)).
					SetAccuracy(getRandomStat(ea.Accuracy(), 5)).
					SetAvoidability(getRandomStat(ea.Avoidability(), 5)).
					SetHands(getRandomStat(ea.Hands(), 5)).
					SetSpeed(getRandomStat(ea.Speed(), 5)).
					SetJump(getRandomStat(ea.Jump(), 5)).
					SetSlots(ea.Slots())
			case inventory.TypeValueUse, inventory.TypeValueSetup, inventory.TypeValueETC:
				b.SetQuantity(quantity).
					SetOwnerId(ownerId).
					SetFlag(flag).
					SetRechargeable(rechargeable)
			case inventory.TypeValueCash:
				if item.GetClassification(item.Id(templateId)) == item.ClassificationPet {
					pe, err := p.petProcessor.Create(characterId, templateId)
					if err != nil {
						return err
					}
					b.SetPetId(pe.Id()).
						SetCashId(pe.CashId()).
						SetOwnerId(pe.OwnerId()).
						SetFlag(pe.Flag()).
						SetExpiration(pe.Expiration()).
						SetPurchaseBy(pe.PurchaseBy())
				} else {
					b.SetQuantity(quantity).
						SetOwnerId(ownerId).
						SetFlag(flag)
				}
			}

			var err error
			a, err = create(tx, p.t.Id(), b.Build())
			if err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, CreatedEventStatusProvider(transactionId, characterId, a))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return a, nil
	}
}

func (p *Processor) CreateFromModel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, m Model) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, m Model) (Model, error) {
		p.l.Debugf("Character [%d] creating asset from model for template [%d] in compartment [%s].", characterId, m.TemplateId(), m.CompartmentId().String())
		var a Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var err error
			a, err = create(tx, p.t.Id(), m)
			if err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, CreatedEventStatusProvider(transactionId, characterId, a))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return a, nil
	}
}

func (p *Processor) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, slot int16, m Model) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, slot int16, m Model) (Model, error) {
		p.l.Debugf("Character [%d] attempting to accept asset template [%d] in slot [%d] of compartment [%s].", characterId, m.TemplateId(), slot, compartmentId.String())
		var a Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			nm := Clone(m).
				SetCompartmentId(compartmentId).
				SetSlot(slot).
				SetCreatedAt(time.Now()).
				Build()

			var err error
			a, err = create(tx, p.t.Id(), nm)
			if err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, AcceptedEventStatusProvider(transactionId, characterId, a))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return a, nil
	}
}

func (p *Processor) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a Model) error {
		return func(a Model) error {
			p.l.Debugf("Attempting to release asset [%d].", a.Id())
			err := deleteById(p.db, p.t.Id(), a.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to release asset [%d].", a.Id())
				return err
			}
			p.l.Debugf("Released asset [%d].", a.Id())
			return mb.Put(asset.EnvEventTopicStatus, ReleasedEventStatusProvider(transactionId, characterId, a))
		}
	}
}

func getRandomStat(defaultValue uint16, max uint16) uint16 {
	if defaultValue == 0 {
		return 0
	}
	maxRange := math.Min(math.Ceil(float64(defaultValue)*0.1), float64(max))
	return uint16(float64(defaultValue)-maxRange) + uint16(math.Floor(rand.Float64()*(maxRange*2.0+1.0)))
}
