package snapshot

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/inventory"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Fetch seams: the exact per-component REST providers today's decorator
// chain uses (character/processor.go GetById + InventoryDecorator +
// SkillModelDecorator; character/buff/processor.go). Package-level vars so
// tests prove zero-REST behavior without HTTP fakes (project precedent:
// monsterByIdFn in movement, task-120).
var (
	coreFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
		return character.NewProcessor(l, ctx).GetById()(characterId)
	}
	inventoryFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (inventory.Model, error) {
		return inventory.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
	skillsFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]skill.Model, error) {
		return skill.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
	buffsFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]buff.Model, error) {
		return buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
)

// Processor is the snapshot's read API (FR-3.5): any handler can resolve a
// locally-sessioned character's decorated model or active buffs from
// in-process state, with per-component REST miss-fallback that backfills.
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// Get returns the composed decorated character model — the same shape
// cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator) returns today.
// Fast path: all components valid → in-process composition. Slow path:
// only invalid components are refetched over REST (counted), backfilled
// under a generation check, and the fetched values are served to this
// caller even if a concurrent event discarded the backfill.
func (p *Processor) Get(characterId uint32) (character.Model, error) {
	r := GetRegistry()
	if m, ok := r.ComposedIfValid(p.t, characterId); ok {
		recordRead(p.t, componentCore, outcomeHit)
		recordRead(p.t, componentSkills, outcomeHit)
		recordRead(p.t, componentInventory, outcomeHit)
		return m, nil
	}

	v := r.View(p.t, characterId)

	core := v.Core
	if v.CoreValid {
		recordRead(p.t, componentCore, outcomeHit)
	} else {
		recordRead(p.t, componentCore, outcomeMiss)
		fetched, err := coreFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			recordRead(p.t, componentCore, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot core fallback failed for character [%d].", characterId)
			return character.Model{}, err
		}
		recordRead(p.t, componentCore, outcomeFallbackSuccess)
		p.l.Debugf("Snapshot fallback: core refetched for character [%d].", characterId)
		r.BackfillCore(p.t, characterId, fetched, v.CoreGen)
		core = fetched
	}

	inv := v.Inv
	invOk := v.InvValid
	if v.InvValid {
		recordRead(p.t, componentInventory, outcomeHit)
	} else {
		recordRead(p.t, componentInventory, outcomeMiss)
		fetched, err := inventoryFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			// Matches today's character.ProcessorImpl.InventoryDecorator: a
			// fallback failure degrades the model (inventory left as-is)
			// rather than failing the whole read.
			recordRead(p.t, componentInventory, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot inventory fallback failed for character [%d]; serving degraded model.", characterId)
		} else {
			recordRead(p.t, componentInventory, outcomeFallbackSuccess)
			p.l.Debugf("Snapshot fallback: inventory refetched for character [%d].", characterId)
			r.BackfillInventory(p.t, characterId, fetched, v.InvGen)
			inv = fetched
			invOk = true
		}
	}

	skills := v.Skills
	skillsOk := v.SkillsValid
	if v.SkillsValid {
		recordRead(p.t, componentSkills, outcomeHit)
	} else {
		recordRead(p.t, componentSkills, outcomeMiss)
		fetched, err := skillsFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			// Matches today's character.ProcessorImpl.SkillModelDecorator: a
			// fallback failure degrades the model (skills left as-is) rather
			// than failing the whole read.
			recordRead(p.t, componentSkills, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot skills fallback failed for character [%d]; serving degraded model.", characterId)
		} else {
			recordRead(p.t, componentSkills, outcomeFallbackSuccess)
			p.l.Debugf("Snapshot fallback: skills refetched for character [%d].", characterId)
			r.BackfillSkills(p.t, characterId, fetched, v.SkillsGen)
			skills = fetched
			skillsOk = true
		}
	}

	m := core
	if v.PosValid {
		m = character.CloneModel(m).SetX(v.PosX).SetY(v.PosY).MustBuild()
	}
	// Only decorate with a component that has ever been successfully
	// populated. On a first-ever fallback failure there is no prior value
	// to fall back to, and SetInventory/SetSkills require a real component
	// value (a zero-value inventory.Model is not buildable) — exactly the
	// situation today's decorator chain avoids by leaving the undecorated
	// core model untouched.
	if invOk {
		m = m.SetInventory(inv)
	}
	if skillsOk {
		m = m.SetSkills(skills)
	}
	return m, nil
}

// GetBuffs returns the character's active buffs from the snapshot,
// lazy-seeding via REST on miss. Reads self-filter entries past their
// ExpiresAt (bounds the atlas-buffs-restart residual, event-coverage §5).
func (p *Processor) GetBuffs(characterId uint32) ([]buff.Model, error) {
	r := GetRegistry()
	v := r.View(p.t, characterId)
	if v.BuffsValid {
		recordRead(p.t, componentBuffs, outcomeHit)
		return filterActive(v.Buffs), nil
	}
	recordRead(p.t, componentBuffs, outcomeMiss)
	fetched, err := buffsFetchFn(p.l, p.ctx, characterId)
	if err != nil {
		recordRead(p.t, componentBuffs, outcomeFallbackFailure)
		return nil, err
	}
	recordRead(p.t, componentBuffs, outcomeFallbackSuccess)
	p.l.Debugf("Snapshot fallback: buffs refetched for character [%d].", characterId)
	r.BackfillBuffs(p.t, characterId, fetched, v.BuffsGen)
	return filterActive(fetched), nil
}

// BuffsProvider adapts GetBuffs to the provider shape used across the
// codebase (FR-3.5 naming per design §4.2).
func (p *Processor) BuffsProvider(characterId uint32) model.Provider[[]buff.Model] {
	return func() ([]buff.Model, error) {
		return p.GetBuffs(characterId)
	}
}

func filterActive(bs []buff.Model) []buff.Model {
	out := make([]buff.Model, 0, len(bs))
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		out = append(out, b)
	}
	return out
}
