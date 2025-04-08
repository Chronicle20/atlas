package asset

import (
	"atlas-inventory/cash"
	"atlas-inventory/equipable"
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/asset"
	asset2 "atlas-inventory/kafka/producer/asset"
	"atlas-inventory/pet"
	"atlas-inventory/stackable"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:   p.l,
		ctx: p.ctx,
		db:  db,
	}
}

func (p *Processor) GetByCompartmentId(compartmentId uuid.UUID, inventoryType inventory.Type) ([]Model[any], error) {
	return p.ByCompartmentIdProvider(compartmentId, inventoryType)()
}

func (p *Processor) ByCompartmentIdProvider(compartmentId uuid.UUID, inventoryType inventory.Type) model.Provider[[]Model[any]] {
	t := tenant.MustFromContext(p.ctx)
	ap := model.SliceMap(Make)(getByCompartmentId(t.Id(), compartmentId)(p.db))(model.ParallelMap())
	var refDecorator model.Transformer[Model[any], Model[any]]
	if inventoryType == inventory.TypeValueEquip {
		refDecorator = p.DecorateEquipable
	} else if inventoryType == inventory.TypeValueUse || inventoryType == inventory.TypeValueSetup || inventoryType == inventory.TypeValueETC {
		sm, err := model.CollectToMap(stackable.ByCompartmentIdProvider(p.l)(p.ctx)(p.db)(compartmentId), stackable.Identity, stackable.This)()
		if err != nil {
			return model.ErrorProvider[[]Model[any]](err)
		}
		refDecorator = p.DecorateStackable(sm)
	} else if inventoryType == inventory.TypeValueCash {
		refDecorator = p.DecorateCash
	}
	if refDecorator == nil {
		p.l.Errorf("Unable to decorate assets in compartment [%s]. This will lead to unexpected behavior.", compartmentId.String())
		return ap
	}
	return model.SliceMap(refDecorator)(ap)(model.ParallelMap())
}

func (p *Processor) DecorateEquipable(m Model[any]) (Model[any], error) {
	e, err := equipable.GetById(p.l)(p.ctx)(m.ReferenceId())
	if err != nil {
		return Model[any]{}, nil
	}
	return Clone(m).
		SetReferenceData(EquipableReferenceData{
			strength:       e.Strength(),
			dexterity:      e.Dexterity(),
			intelligence:   e.Intelligence(),
			luck:           e.Luck(),
			hp:             e.HP(),
			mp:             e.MP(),
			weaponAttack:   e.WeaponAttack(),
			magicAttack:    e.MagicAttack(),
			weaponDefense:  e.WeaponDefense(),
			magicDefense:   e.MagicDefense(),
			accuracy:       e.Accuracy(),
			avoidability:   e.Avoidability(),
			hands:          e.Hands(),
			speed:          e.Speed(),
			jump:           e.Jump(),
			slots:          e.Slots(),
			ownerName:      e.OwnerName(),
			locked:         e.Locked(),
			spikes:         e.Spikes(),
			karmaUsed:      e.KarmaUsed(),
			cold:           e.Cold(),
			canBeTraded:    e.CanBeTraded(),
			levelType:      e.LevelType(),
			level:          e.Level(),
			experience:     e.Experience(),
			hammersApplied: e.HammersApplied(),
			expiration:     e.Expiration(),
		}).
		Build(), nil
}

func (p *Processor) DecorateStackable(sm map[uint32]stackable.Model) model.Transformer[Model[any], Model[any]] {
	return func(m Model[any]) (Model[any], error) {
		var s stackable.Model
		var ok bool
		if s, ok = sm[m.ReferenceId()]; !ok {
			return m, errors.New("cannot locate reference")
		}

		var rd any
		if m.ReferenceType() == ReferenceTypeConsumable {
			rd = ConsumableReferenceData{
				quantity:     s.Quantity(),
				owner:        s.Owner(),
				flag:         s.Flag(),
				rechargeable: s.Rechargeable(),
			}
		} else if m.ReferenceType() == ReferenceTypeSetup {
			rd = SetupReferenceData{
				quantity: s.Quantity(),
				owner:    s.Owner(),
				flag:     s.Flag(),
			}
		} else if m.ReferenceType() == ReferenceTypeEtc {
			rd = EtcReferenceData{
				quantity: s.Quantity(),
				owner:    s.Owner(),
				flag:     s.Flag(),
			}
		}

		return Clone(m).
			SetReferenceData(rd).
			Build(), nil
	}
}

func (p *Processor) DecorateCash(m Model[any]) (Model[any], error) {
	if m.ReferenceType() == ReferenceTypeCash {
		ci, err := cash.GetById(p.l)(p.ctx)(m.ReferenceId())
		if err != nil {
			return m, errors.New("cannot locate reference")
		}
		return Clone(m).
			SetReferenceData(CashReferenceData{
				quantity:   ci.Quantity(),
				owner:      ci.Owner(),
				flag:       ci.Flag(),
				purchaseBy: ci.PurchasedBy(),
			}).
			Build(), nil
	} else if m.ReferenceType() == ReferenceTypePet {
		ci, err := cash.GetById(p.l)(p.ctx)(m.ReferenceId())
		if err != nil {
			return m, errors.New("cannot locate reference")
		}
		pi, err := pet.GetById(p.l)(p.ctx)(m.ReferenceId())
		if err != nil {
			return m, errors.New("cannot locate reference")
		}
		return Clone(m).
			SetReferenceData(PetReferenceData{
				cashId:     ci.CashId(),
				owner:      ci.Owner(),
				flag:       ci.Flag(),
				purchaseBy: ci.PurchasedBy(),
				name:       pi.Name(),
				level:      pi.Level(),
				closeness:  pi.Closeness(),
				fullness:   pi.Fullness(),
				expiration: pi.Expiration(),
			}).
			Build(), nil
	}
	return m, nil
}

func (p *Processor) Delete(mb *message.Buffer) func(a Model[any]) error {
	return func(a Model[any]) error {
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Attempting to delete asset [%d].", a.Id())
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			var deleteRefFunc func(id uint32) error
			if a.ReferenceType() == ReferenceTypeEquipable {
				deleteRefFunc = equipable.Delete(p.l)(p.ctx)
			} else if a.ReferenceType() == ReferenceTypeConsumable || a.ReferenceType() == ReferenceTypeSetup || a.ReferenceType() == ReferenceTypeEtc {
				deleteRefFunc = stackable.Delete(p.l)(p.ctx)(tx)
			} else if a.ReferenceType() == ReferenceTypeCash {
				// TODO
			} else if a.ReferenceType() == ReferenceTypePet {
				// TODO
			}

			if deleteRefFunc == nil {
				p.l.Errorf("Unable to locate delete function for asset [%d]. This will lead to a dangling asset.", a.Id())
				return nil
			}
			err := deleteRefFunc(a.ReferenceId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete asset [%d], due to error deleting reference [%d].", a.Id(), a.ReferenceId())
				return err
			}
			err = deleteById(tx, t.Id(), a.Id())
			if err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, asset2.DeletedEventStatusProvider(a.Id()))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete asset [%d].", a.Id())
			return txErr
		}
		p.l.Debugf("Deleted asset [%d].", a.Id())
		return nil
	}
}
