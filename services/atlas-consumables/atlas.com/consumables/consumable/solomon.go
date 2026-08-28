package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/compartment"
	consumable3 "atlas-consumables/data/consumable"
	character2 "atlas-consumables/map/character"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// routesToSolomon reports whether itemId is a Writ of Solomon (is_exp_up_item,
// classification 237) and therefore routes to ConsumeSolomon.
func routesToSolomon(itemId item2.Id) bool {
	return item2.GetClassification(itemId) == item2.ClassificationConsumableExpUpItem
}

// solomonDeps is the collaborator set consumeSolomon needs.
//
// Split out from the exported consumer so the ordering contract — every
// fallible read BEFORE the commit, the eligibility rules gating the commit —
// is pinnable with the package's existing mocks. ConsumeSolomon below binds
// the real processors.
type solomonDeps struct {
	data        consumable3.Processor
	fields      character2.Processor
	compartment compartment.Processor
	character   character.Processor
	// onError releases the reservation and notifies the client. Bound to
	// ProcessorImpl.ConsumeError with the USE compartment already applied.
	onError func(err error) error
}

// consumeSolomon applies a Writ of Solomon: read, validate eligibility,
// commit, then credit the banked EXP.
//
// Ordering mirrors consumeMorphCoupon/ConsumeStandard. Every fallible read
// happens before ConsumeItem so a data failure returns the Writ to the player
// (FR-6). The eligibility checks (spec/exp present and positive, character
// level within maxLevel, stored balance zero) all reject BEFORE the commit
// too, so a rejected Writ is never destroyed.
func consumeSolomon(l logrus.FieldLogger, ctx context.Context, d solomonDeps, transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) error {
	pg, _ := model.NewGroup(ctx)
	ff := model.Submit(pg, func() (field.Model, error) { return d.fields.GetMap(characterId) })
	fi := model.Submit(pg, func() (consumable3.Model, error) { return d.data.GetById(uint32(itemId)) })
	fc := model.Submit(pg, func() (character.Model, error) { return d.character.GetById()(characterId) })
	if err := pg.Wait(); err != nil {
		return d.onError(err)
	}
	f, ci, c := ff.Get(), fi.Get(), fc.Get()

	amount, ok := ci.GetSpec(consumable3.SpecTypeExperience)
	if !ok || amount <= 0 {
		l.Warnf("Character [%d] consumed Writ of Solomon [%d] but its spec/exp is absent or non-positive; the tenant's Item.wz likely predates the spec/exp parse.", characterId, itemId)
		return d.onError(errors.New("writ of solomon has no spec/exp"))
	}

	if ci.MaxLevel() > 0 && uint32(c.Level()) > ci.MaxLevel() {
		l.Warnf("Character [%d] level [%d] exceeds Writ of Solomon [%d] maxLevel [%d]; rejecting per the item's level gate.", characterId, c.Level(), itemId, ci.MaxLevel())
		return d.onError(errors.New("character level exceeds writ of solomon maxLevel"))
	}

	if c.GachaponExperience() != 0 {
		l.Warnf("Character [%d] has a non-zero stored EXP balance [%d]; rejecting Writ of Solomon [%d] until the banked balance is spent.", characterId, c.GachaponExperience(), itemId)
		return d.onError(errors.New("character already has a non-zero stored exp balance"))
	}

	if err := d.compartment.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot); err != nil {
		return d.onError(err)
	}

	if err := d.character.CreditStoredExperience(f, characterId, uint32(amount), "SOLOMON_ITEM"); err != nil {
		l.WithError(err).Errorf("Character [%d] consumed Writ of Solomon [%d] but crediting stored EXP of [%d] failed.", characterId, itemId, amount)
	}
	return nil
}

// ConsumeSolomon commits a reserved Writ of Solomon (classification 237) from
// the USE compartment and credits its banked EXP to the character.
func ConsumeSolomon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			d := solomonDeps{
				data:        consumable3.NewProcessor(l, ctx),
				fields:      character2.NewProcessor(l, ctx),
				compartment: compartment.NewProcessor(l, ctx),
				character:   character.NewProcessor(l, ctx),
				onError: func(err error) error {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
				},
			}
			return consumeSolomon(l, ctx, d, transactionId, characterId, slot, itemId)
		}
	}
}
